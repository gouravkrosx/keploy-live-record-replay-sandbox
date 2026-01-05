package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

type InventoryHandler struct {
	db *database.Database
}

func NewInventoryHandler(db *database.Database) *InventoryHandler {
	return &InventoryHandler{db: db}
}

// ListInventory returns inventory items
func (h *InventoryHandler) ListInventory(w http.ResponseWriter, r *http.Request) {
	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	var inventory []models.Inventory
	var total int64

	query := h.db.Model(&models.Inventory{})

	// Apply filters
	if productID := r.URL.Query().Get("productId"); productID != "" {
		if pID, err := uuid.Parse(productID); err == nil {
			query = query.Where("product_id = ?", pID)
		}
	}
	if warehouseID := r.URL.Query().Get("warehouseId"); warehouseID != "" {
		if wID, err := uuid.Parse(warehouseID); err == nil {
			query = query.Where("warehouse_id = ?", wID)
		}
	}
	if lowStock := r.URL.Query().Get("lowStock"); lowStock == "true" {
		query = query.Where("quantity - reserved_qty <= reorder_level")
	}

	query.Count(&total)
	query.Offset(offset).Limit(limit).Find(&inventory)

	// Enrich with product info
	type inventoryWithProduct struct {
		models.Inventory
		Product models.Product `json:"product"`
	}

	enriched := make([]inventoryWithProduct, len(inventory))
	for i, inv := range inventory {
		var product models.Product
		h.db.First(&product, inv.ProductID)
		enriched[i] = inventoryWithProduct{
			Inventory: inv,
			Product:   product,
		}
	}

	utils.PaginatedJSONResponse(w, enriched, page, limit, int(total))
}

// UpdateInventory updates inventory levels
func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	inventoryID, err := uuid.Parse(chi.URLParam(r, "inventoryId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid inventory ID format")
		return
	}

	var updates map[string]interface{}
	if err := utils.ParseJSON(r, &updates); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Only allow certain fields
	allowedFields := map[string]bool{"quantity": true, "reorderLevel": true, "reorderQuantity": true}
	for key := range updates {
		if !allowedFields[key] {
			delete(updates, key)
		}
	}

	if err := h.db.Model(&models.Inventory{}).Where("id = ?", inventoryID).Updates(updates).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update inventory")
		return
	}

	var inventory models.Inventory
	h.db.First(&inventory, inventoryID)
	utils.JSONResponse(w, http.StatusOK, inventory)
}

// TransferInventory transfers stock between warehouses
func (h *InventoryHandler) TransferInventory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductID       string `json:"productId"`
		FromWarehouseID string `json:"fromWarehouseId"`
		ToWarehouseID   string `json:"toWarehouseId"`
		Quantity        int    `json:"quantity"`
		Note            string `json:"note"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.ProductID == "" || req.FromWarehouseID == "" || req.ToWarehouseID == "" || req.Quantity < 1 {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "All fields are required")
		return
	}

	productID, _ := uuid.Parse(req.ProductID)
	fromWarehouseID, _ := uuid.Parse(req.FromWarehouseID)
	toWarehouseID, _ := uuid.Parse(req.ToWarehouseID)

	tx := h.db.Begin()

	// Get source inventory
	var fromInventory models.Inventory
	if err := tx.Where("product_id = ? AND warehouse_id = ?", productID, fromWarehouseID).First(&fromInventory).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Source inventory not found")
		return
	}

	availableQty := fromInventory.Quantity - fromInventory.ReservedQty
	if availableQty < req.Quantity {
		tx.Rollback()
		utils.ErrorResponse(w, http.StatusBadRequest, "INSUFFICIENT_STOCK", "Not enough stock to transfer")
		return
	}

	// Update source
	fromInventory.Quantity -= req.Quantity
	tx.Save(&fromInventory)

	// Get or create destination inventory
	var toInventory models.Inventory
	if err := tx.Where("product_id = ? AND warehouse_id = ?", productID, toWarehouseID).First(&toInventory).Error; err != nil {
		toInventory = models.Inventory{
			ProductID:   productID,
			WarehouseID: toWarehouseID,
			Quantity:    req.Quantity,
		}
		tx.Create(&toInventory)
	} else {
		toInventory.Quantity += req.Quantity
		tx.Save(&toInventory)
	}

	tx.Commit()

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":       "Transfer completed successfully",
		"fromInventory": fromInventory,
		"toInventory":   toInventory,
	})
}

// AdjustInventory performs bulk inventory adjustments
func (h *InventoryHandler) AdjustInventory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason      string `json:"reason"`
		Adjustments []struct {
			ProductID   string `json:"productId"`
			WarehouseID string `json:"warehouseId"`
			Quantity    int    `json:"quantity"`
		} `json:"adjustments"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	tx := h.db.Begin()
	adjusted := 0

	for _, adj := range req.Adjustments {
		productID, _ := uuid.Parse(adj.ProductID)
		warehouseID, _ := uuid.Parse(adj.WarehouseID)

		var inventory models.Inventory
		if err := tx.Where("product_id = ? AND warehouse_id = ?", productID, warehouseID).First(&inventory).Error; err != nil {
			// Create new inventory record
			inventory = models.Inventory{
				ProductID:   productID,
				WarehouseID: warehouseID,
				Quantity:    adj.Quantity,
			}
			tx.Create(&inventory)
		} else {
			inventory.Quantity += adj.Quantity
			if inventory.Quantity < 0 {
				inventory.Quantity = 0
			}
			tx.Save(&inventory)
		}
		adjusted++
	}

	tx.Commit()

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":  "Adjustments applied successfully",
		"adjusted": adjusted,
		"reason":   req.Reason,
	})
}
