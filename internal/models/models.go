package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base model with UUID primary key
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:char(36);primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// User represents a customer, seller, or admin
type User struct {
	BaseModel
	Email         string `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password      string `gorm:"size:255;not null" json:"-"`
	FirstName     string `gorm:"size:50;not null" json:"firstName"`
	LastName      string `gorm:"size:50;not null" json:"lastName"`
	Phone         string `gorm:"size:20" json:"phone,omitempty"`
	Role          string `gorm:"size:20;default:customer" json:"role"`
	Status        string `gorm:"size:20;default:active" json:"status"`
	EmailVerified bool   `gorm:"default:false" json:"emailVerified"`

	// Relations
	Addresses []Address `gorm:"foreignKey:UserID" json:"addresses,omitempty"`
	Orders    []Order   `gorm:"foreignKey:UserID" json:"orders,omitempty"`
	Reviews   []Review  `gorm:"foreignKey:UserID" json:"reviews,omitempty"`
	Cart      *Cart     `gorm:"foreignKey:UserID" json:"cart,omitempty"`
	Products  []Product `gorm:"foreignKey:SellerID" json:"products,omitempty"` // For sellers
}

// Category represents product categories (hierarchical)
type Category struct {
	BaseModel
	Name        string     `gorm:"size:100;not null" json:"name"`
	Slug        string     `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	ParentID    *uuid.UUID `gorm:"type:char(36);index" json:"parentId,omitempty"`
	Image       string     `gorm:"size:500" json:"image,omitempty"`
	SortOrder   int        `gorm:"default:0" json:"sortOrder"`

	// Relations
	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

// Product represents items for sale
type Product struct {
	BaseModel
	Name           string    `gorm:"size:255;not null" json:"name"`
	Slug           string    `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Description    string    `gorm:"type:text" json:"description,omitempty"`
	Price          float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	CompareAtPrice float64   `gorm:"type:decimal(10,2)" json:"compareAtPrice,omitempty"`
	SKU            string    `gorm:"size:100;index" json:"sku,omitempty"`
	CategoryID     uuid.UUID `gorm:"type:char(36);index;not null" json:"categoryId"`
	SellerID       uuid.UUID `gorm:"type:char(36);index;not null" json:"sellerId"`
	Images         string    `gorm:"type:json" json:"images,omitempty"`     // JSON array of image URLs
	Attributes     string    `gorm:"type:json" json:"attributes,omitempty"` // JSON object
	Status         string    `gorm:"size:20;default:draft" json:"status"`

	// Relations
	Category  Category    `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Seller    User        `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Reviews   []Review    `gorm:"foreignKey:ProductID" json:"reviews,omitempty"`
	Inventory []Inventory `gorm:"foreignKey:ProductID" json:"inventory,omitempty"`
}

// Order represents customer purchases
type Order struct {
	BaseModel
	OrderNumber       string     `gorm:"size:50;uniqueIndex;not null" json:"orderNumber"`
	UserID            uuid.UUID  `gorm:"type:char(36);index;not null" json:"userId"`
	Status            string     `gorm:"size:20;default:pending" json:"status"`
	Subtotal          float64    `gorm:"type:decimal(10,2);not null" json:"subtotal"`
	Discount          float64    `gorm:"type:decimal(10,2);default:0" json:"discount"`
	ShippingCost      float64    `gorm:"type:decimal(10,2);default:0" json:"shippingCost"`
	Tax               float64    `gorm:"type:decimal(10,2);default:0" json:"tax"`
	Total             float64    `gorm:"type:decimal(10,2);not null" json:"total"`
	CouponID          *uuid.UUID `gorm:"type:char(36);index" json:"couponId,omitempty"`
	ShippingAddressID uuid.UUID  `gorm:"type:char(36);not null" json:"shippingAddressId"`
	BillingAddressID  *uuid.UUID `gorm:"type:char(36)" json:"billingAddressId,omitempty"`
	Notes             string     `gorm:"type:text" json:"notes,omitempty"`

	// Relations
	User            User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Items           []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Payment         *Payment    `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
	Coupon          *Coupon     `gorm:"foreignKey:CouponID" json:"coupon,omitempty"`
	ShippingAddress Address     `gorm:"foreignKey:ShippingAddressID" json:"shippingAddress,omitempty"`
	BillingAddress  *Address    `gorm:"foreignKey:BillingAddressID" json:"billingAddress,omitempty"`
}

// OrderItem represents individual items in an order
type OrderItem struct {
	BaseModel
	OrderID      uuid.UUID `gorm:"type:char(36);index;not null" json:"orderId"`
	ProductID    uuid.UUID `gorm:"type:char(36);index;not null" json:"productId"`
	ProductName  string    `gorm:"size:255;not null" json:"productName"`
	ProductImage string    `gorm:"size:500" json:"productImage,omitempty"`
	SKU          string    `gorm:"size:100" json:"sku,omitempty"`
	Quantity     int       `gorm:"not null" json:"quantity"`
	UnitPrice    float64   `gorm:"type:decimal(10,2);not null" json:"unitPrice"`
	TotalPrice   float64   `gorm:"type:decimal(10,2);not null" json:"totalPrice"`

	// Relations
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// Review represents product reviews
type Review struct {
	BaseModel
	ProductID        uuid.UUID  `gorm:"type:char(36);index;not null" json:"productId"`
	UserID           uuid.UUID  `gorm:"type:char(36);index;not null" json:"userId"`
	OrderID          *uuid.UUID `gorm:"type:char(36);index" json:"orderId,omitempty"`
	Rating           int        `gorm:"not null" json:"rating"`
	Title            string     `gorm:"size:255" json:"title,omitempty"`
	Content          string     `gorm:"type:text" json:"content,omitempty"`
	Images           string     `gorm:"type:json" json:"images,omitempty"`
	VerifiedPurchase bool       `gorm:"default:false" json:"verifiedPurchase"`
	HelpfulCount     int        `gorm:"default:0" json:"helpfulCount"`

	// Relations
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Address represents user addresses
type Address struct {
	BaseModel
	UserID       uuid.UUID `gorm:"type:char(36);index;not null" json:"userId"`
	Type         string    `gorm:"size:20;default:both" json:"type"` // shipping, billing, both
	FirstName    string    `gorm:"size:50;not null" json:"firstName"`
	LastName     string    `gorm:"size:50;not null" json:"lastName"`
	Company      string    `gorm:"size:100" json:"company,omitempty"`
	AddressLine1 string    `gorm:"size:255;not null" json:"addressLine1"`
	AddressLine2 string    `gorm:"size:255" json:"addressLine2,omitempty"`
	City         string    `gorm:"size:100;not null" json:"city"`
	State        string    `gorm:"size:100;not null" json:"state"`
	PostalCode   string    `gorm:"size:20;not null" json:"postalCode"`
	Country      string    `gorm:"size:100;not null" json:"country"`
	Phone        string    `gorm:"size:20" json:"phone,omitempty"`
	IsDefault    bool      `gorm:"default:false" json:"isDefault"`
}

// Payment represents payment transactions
type Payment struct {
	BaseModel
	OrderID         uuid.UUID  `gorm:"type:char(36);uniqueIndex;not null" json:"orderId"`
	Amount          float64    `gorm:"type:decimal(10,2);not null" json:"amount"`
	Currency        string     `gorm:"size:3;default:USD" json:"currency"`
	Method          string     `gorm:"size:20;not null" json:"method"` // card, bank_transfer, wallet, cod
	Status          string     `gorm:"size:20;default:pending" json:"status"`
	TransactionID   string     `gorm:"size:255" json:"transactionId,omitempty"`
	GatewayResponse string     `gorm:"type:json" json:"gatewayResponse,omitempty"`
	RefundedAmount  float64    `gorm:"type:decimal(10,2);default:0" json:"refundedAmount"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
}

// Cart represents shopping carts
type Cart struct {
	BaseModel
	UserID   uuid.UUID  `gorm:"type:char(36);uniqueIndex;not null" json:"userId"`
	CouponID *uuid.UUID `gorm:"type:char(36);index" json:"couponId,omitempty"`

	// Relations
	Items  []CartItem `gorm:"foreignKey:CartID" json:"items,omitempty"`
	Coupon *Coupon    `gorm:"foreignKey:CouponID" json:"coupon,omitempty"`
}

// CartItem represents items in shopping carts
type CartItem struct {
	BaseModel
	CartID    uuid.UUID `gorm:"type:char(36);index;not null" json:"cartId"`
	ProductID uuid.UUID `gorm:"type:char(36);index;not null" json:"productId"`
	Quantity  int       `gorm:"not null;default:1" json:"quantity"`

	// Relations
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// Inventory represents stock per warehouse
type Inventory struct {
	BaseModel
	ProductID       uuid.UUID  `gorm:"type:char(36);index;not null" json:"productId"`
	WarehouseID     uuid.UUID  `gorm:"type:char(36);index;not null" json:"warehouseId"`
	WarehouseName   string     `gorm:"size:100" json:"warehouseName"`
	Quantity        int        `gorm:"not null;default:0" json:"quantity"`
	ReservedQty     int        `gorm:"not null;default:0" json:"reservedQuantity"`
	ReorderLevel    int        `gorm:"default:10" json:"reorderLevel"`
	ReorderQuantity int        `gorm:"default:50" json:"reorderQuantity"`
	LastRestocked   *time.Time `json:"lastRestocked,omitempty"`
}

// Coupon represents promotional discounts
type Coupon struct {
	BaseModel
	Code                 string     `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Description          string     `gorm:"type:text" json:"description,omitempty"`
	Type                 string     `gorm:"size:20;not null" json:"type"` // percentage, fixed, free_shipping
	Value                float64    `gorm:"type:decimal(10,2);not null" json:"value"`
	MinOrderAmount       float64    `gorm:"type:decimal(10,2)" json:"minOrderAmount,omitempty"`
	MaxDiscount          float64    `gorm:"type:decimal(10,2)" json:"maxDiscount,omitempty"`
	UsageLimit           int        `json:"usageLimit,omitempty"`
	UsageCount           int        `gorm:"default:0" json:"usageCount"`
	PerUserLimit         int        `json:"perUserLimit,omitempty"`
	ApplicableCategories string     `gorm:"type:json" json:"applicableCategories,omitempty"`
	ApplicableProducts   string     `gorm:"type:json" json:"applicableProducts,omitempty"`
	StartDate            *time.Time `json:"startDate,omitempty"`
	EndDate              *time.Time `json:"endDate,omitempty"`
	IsActive             bool       `gorm:"default:true" json:"isActive"`
}

// RefreshToken stores refresh tokens for JWT
type RefreshToken struct {
	BaseModel
	UserID    uuid.UUID `gorm:"type:char(36);index;not null" json:"userId"`
	Token     string    `gorm:"size:500;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
	Revoked   bool      `gorm:"default:false" json:"revoked"`
}
