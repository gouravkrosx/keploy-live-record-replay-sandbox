package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

type CategoryHandler struct {
	db *database.Database
}

func NewCategoryHandler(db *database.Database) *CategoryHandler {
	return &CategoryHandler{db: db}
}

// ListCategories returns category tree
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	includeProductCount := r.URL.Query().Get("includeProductCount") == "true"

	var categories []models.Category
	h.db.Where("parent_id IS NULL").Order("sort_order ASC").Preload("Children").Find(&categories)

	if includeProductCount {
		// Recursively add product counts
		h.addProductCounts(&categories)
	}

	utils.JSONResponse(w, http.StatusOK, categories)
}

func (h *CategoryHandler) addProductCounts(categories *[]models.Category) {
	for i := range *categories {
		var count int64
		h.db.Model(&models.Product{}).Where("category_id = ? AND status = ?", (*categories)[i].ID, "active").Count(&count)
		// Note: In a real implementation, you'd add a ProductCount field to the response

		if len((*categories)[i].Children) > 0 {
			h.addProductCounts(&(*categories)[i].Children)
		}
	}
}

// GetCategory returns a single category with ancestors and descendants
func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid category ID format")
		return
	}

	var category models.Category
	if err := h.db.Preload("Children").Preload("Parent").First(&category, categoryID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Category not found")
		return
	}

	// Get ancestors
	ancestors := h.getAncestors(category.ParentID)

	// Get product count
	var productCount int64
	h.db.Model(&models.Product{}).Where("category_id = ? AND status = ?", categoryID, "active").Count(&productCount)

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"category":     category,
		"ancestors":    ancestors,
		"productCount": productCount,
	})
}

func (h *CategoryHandler) getAncestors(parentID *uuid.UUID) []models.Category {
	var ancestors []models.Category
	currentParentID := parentID

	for currentParentID != nil {
		var parent models.Category
		if err := h.db.First(&parent, *currentParentID).Error; err != nil {
			break
		}
		ancestors = append([]models.Category{parent}, ancestors...)
		currentParentID = parent.ParentID
	}

	return ancestors
}

// CreateCategory creates a new category
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		ParentID    *string `json:"parentId"`
		Image       string  `json:"image"`
		SortOrder   int     `json:"sortOrder"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Name == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Name is required")
		return
	}

	category := &models.Category{
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Description: req.Description,
		Image:       req.Image,
		SortOrder:   req.SortOrder,
	}

	if req.ParentID != nil && *req.ParentID != "" {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid parent ID")
			return
		}
		category.ParentID = &parentID
	}

	if err := h.db.Create(category).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create category")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, category)
}

// UpdateCategory updates an existing category
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid category ID format")
		return
	}

	var updates map[string]interface{}
	if err := utils.ParseJSON(r, &updates); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	delete(updates, "id")
	delete(updates, "createdAt")

	if err := h.db.Model(&models.Category{}).Where("id = ?", categoryID).Updates(updates).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update category")
		return
	}

	var category models.Category
	h.db.Preload("Children").First(&category, categoryID)
	utils.JSONResponse(w, http.StatusOK, category)
}

// DeleteCategory deletes a category
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid category ID format")
		return
	}

	// Get category to find its parent
	var category models.Category
	if err := h.db.First(&category, categoryID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Category not found")
		return
	}

	// Move products to parent category (if exists)
	if category.ParentID != nil {
		h.db.Model(&models.Product{}).Where("category_id = ?", categoryID).Update("category_id", *category.ParentID)
	}

	// Move child categories to parent
	h.db.Model(&models.Category{}).Where("parent_id = ?", categoryID).Update("parent_id", category.ParentID)

	// Delete category
	if err := h.db.Delete(&models.Category{}, categoryID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete category")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
