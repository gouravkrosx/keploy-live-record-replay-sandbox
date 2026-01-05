package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/marketplace-api/internal/auth"
	"github.com/marketplace-api/internal/config"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/models"
)

func main() {
	log.Println("🌱 Starting data seeder...")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations first
	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Flush all existing data
	log.Println("🗑️  Flushing existing data...")
	flushData(db)

	// Seed data
	log.Println("📊 Seeding fresh data...")
	seedData(db)

	log.Println("✅ Data seeding completed successfully!")
}

func flushData(db *database.Database) {
	// Disable foreign key checks for clean truncation
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")

	// Delete in order of dependencies (children first)
	tables := []string{
		"refresh_tokens",
		"cart_items",
		"carts",
		"order_items",
		"payments",
		"orders",
		"reviews",
		"inventories",
		"products",
		"categories",
		"addresses",
		"coupons",
		"users",
	}

	for _, table := range tables {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
		log.Printf("   Cleared table: %s", table)
	}

	// Re-enable foreign key checks
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

func seedData(db *database.Database) {
	// ==================== USERS ====================
	log.Println("👥 Creating users...")
	users := createUsers(db)

	// ==================== CATEGORIES ====================
	log.Println("📁 Creating categories...")
	categories := createCategories(db)

	// ==================== PRODUCTS ====================
	log.Println("🛍️ Creating products...")
	products := createProducts(db, categories, users)

	// ==================== ADDRESSES ====================
	log.Println("📍 Creating addresses...")
	createAddresses(db, users)

	// ==================== INVENTORY ====================
	log.Println("📦 Creating inventory...")
	createInventory(db, products)

	// ==================== COUPONS ====================
	log.Println("🎟️ Creating coupons...")
	coupons := createCoupons(db)

	// ==================== CARTS ====================
	log.Println("🛒 Creating carts with items...")
	createCartsWithItems(db, users, products)

	// ==================== ORDERS ====================
	log.Println("📋 Creating orders...")
	createOrders(db, users, products, coupons)

	// ==================== REVIEWS ====================
	log.Println("⭐ Creating reviews...")
	createReviews(db, users, products)

	printSummary(db)
}

func createUsers(db *database.Database) []models.User {
	hashedPassword, _ := auth.HashPassword("Password123!")

	users := []models.User{
		// Admin user
		{Email: "admin@marketplace.com", Password: hashedPassword, FirstName: "Admin", LastName: "User", Role: "admin", Status: "active", Phone: "+1-555-0100"},

		// Seller accounts (5)
		{Email: "seller1@marketplace.com", Password: hashedPassword, FirstName: "Tech", LastName: "Store", Role: "seller", Status: "active", Phone: "+1-555-0101"},
		{Email: "seller2@marketplace.com", Password: hashedPassword, FirstName: "Fashion", LastName: "Hub", Role: "seller", Status: "active", Phone: "+1-555-0102"},
		{Email: "seller3@marketplace.com", Password: hashedPassword, FirstName: "Home", LastName: "Essentials", Role: "seller", Status: "active", Phone: "+1-555-0103"},
		{Email: "seller4@marketplace.com", Password: hashedPassword, FirstName: "Sports", LastName: "Plus", Role: "seller", Status: "active", Phone: "+1-555-0104"},
		{Email: "seller5@marketplace.com", Password: hashedPassword, FirstName: "Books", LastName: "World", Role: "seller", Status: "active", Phone: "+1-555-0105"},

		// Regular customers (20 - mix of small and large user base)
		{Email: "john.doe@email.com", Password: hashedPassword, FirstName: "John", LastName: "Doe", Role: "customer", Status: "active", Phone: "+1-555-1001", EmailVerified: true},
		{Email: "jane.smith@email.com", Password: hashedPassword, FirstName: "Jane", LastName: "Smith", Role: "customer", Status: "active", Phone: "+1-555-1002", EmailVerified: true},
		{Email: "mike.johnson@email.com", Password: hashedPassword, FirstName: "Mike", LastName: "Johnson", Role: "customer", Status: "active", Phone: "+1-555-1003", EmailVerified: true},
		{Email: "sarah.williams@email.com", Password: hashedPassword, FirstName: "Sarah", LastName: "Williams", Role: "customer", Status: "active", Phone: "+1-555-1004", EmailVerified: true},
		{Email: "david.brown@email.com", Password: hashedPassword, FirstName: "David", LastName: "Brown", Role: "customer", Status: "active", Phone: "+1-555-1005", EmailVerified: true},
		{Email: "emily.davis@email.com", Password: hashedPassword, FirstName: "Emily", LastName: "Davis", Role: "customer", Status: "active", Phone: "+1-555-1006"},
		{Email: "chris.miller@email.com", Password: hashedPassword, FirstName: "Chris", LastName: "Miller", Role: "customer", Status: "active", Phone: "+1-555-1007"},
		{Email: "lisa.wilson@email.com", Password: hashedPassword, FirstName: "Lisa", LastName: "Wilson", Role: "customer", Status: "active", Phone: "+1-555-1008"},
		{Email: "alex.moore@email.com", Password: hashedPassword, FirstName: "Alex", LastName: "Moore", Role: "customer", Status: "active", Phone: "+1-555-1009"},
		{Email: "rachel.taylor@email.com", Password: hashedPassword, FirstName: "Rachel", LastName: "Taylor", Role: "customer", Status: "active", Phone: "+1-555-1010"},
		{Email: "james.anderson@email.com", Password: hashedPassword, FirstName: "James", LastName: "Anderson", Role: "customer", Status: "active"},
		{Email: "olivia.thomas@email.com", Password: hashedPassword, FirstName: "Olivia", LastName: "Thomas", Role: "customer", Status: "active"},
		{Email: "noah.jackson@email.com", Password: hashedPassword, FirstName: "Noah", LastName: "Jackson", Role: "customer", Status: "active"},
		{Email: "sophia.white@email.com", Password: hashedPassword, FirstName: "Sophia", LastName: "White", Role: "customer", Status: "active"},
		{Email: "liam.harris@email.com", Password: hashedPassword, FirstName: "Liam", LastName: "Harris", Role: "customer", Status: "active"},
		{Email: "ava.martin@email.com", Password: hashedPassword, FirstName: "Ava", LastName: "Martin", Role: "customer", Status: "active"},
		{Email: "william.garcia@email.com", Password: hashedPassword, FirstName: "William", LastName: "Garcia", Role: "customer", Status: "suspended"},
		{Email: "mia.martinez@email.com", Password: hashedPassword, FirstName: "Mia", LastName: "Martinez", Role: "customer", Status: "inactive"},
		{Email: "ethan.robinson@email.com", Password: hashedPassword, FirstName: "Ethan", LastName: "Robinson", Role: "customer", Status: "active"},
		{Email: "isabella.clark@email.com", Password: hashedPassword, FirstName: "Isabella", LastName: "Clark", Role: "customer", Status: "active"},
	}

	for i := range users {
		db.Create(&users[i])
		// Create cart for each user
		cart := &models.Cart{UserID: users[i].ID}
		db.Create(cart)
	}

	log.Printf("   Created %d users", len(users))
	return users
}

func createCategories(db *database.Database) []models.Category {
	categories := []models.Category{}

	// Parent categories
	electronics := models.Category{Name: "Electronics", Slug: "electronics", Description: "Electronic devices and gadgets", SortOrder: 1}
	fashion := models.Category{Name: "Fashion", Slug: "fashion", Description: "Clothing, shoes, and accessories", SortOrder: 2}
	home := models.Category{Name: "Home & Garden", Slug: "home-garden", Description: "Home decor and garden supplies", SortOrder: 3}
	sports := models.Category{Name: "Sports & Outdoors", Slug: "sports-outdoors", Description: "Sports equipment and outdoor gear", SortOrder: 4}
	books := models.Category{Name: "Books & Media", Slug: "books-media", Description: "Books, music, and movies", SortOrder: 5}
	beauty := models.Category{Name: "Beauty & Health", Slug: "beauty-health", Description: "Beauty products and health items", SortOrder: 6}
	toys := models.Category{Name: "Toys & Games", Slug: "toys-games", Description: "Toys, games, and hobbies", SortOrder: 7}
	automotive := models.Category{Name: "Automotive", Slug: "automotive", Description: "Car parts and accessories", SortOrder: 8}

	db.Create(&electronics)
	db.Create(&fashion)
	db.Create(&home)
	db.Create(&sports)
	db.Create(&books)
	db.Create(&beauty)
	db.Create(&toys)
	db.Create(&automotive)

	categories = append(categories, electronics, fashion, home, sports, books, beauty, toys, automotive)

	// Subcategories for Electronics
	subCategories := []models.Category{
		{Name: "Smartphones", Slug: "smartphones", ParentID: &electronics.ID, SortOrder: 1},
		{Name: "Laptops", Slug: "laptops", ParentID: &electronics.ID, SortOrder: 2},
		{Name: "Tablets", Slug: "tablets", ParentID: &electronics.ID, SortOrder: 3},
		{Name: "Audio", Slug: "audio", ParentID: &electronics.ID, SortOrder: 4},
		{Name: "Cameras", Slug: "cameras", ParentID: &electronics.ID, SortOrder: 5},
		{Name: "Gaming", Slug: "gaming", ParentID: &electronics.ID, SortOrder: 6},
		// Fashion subcategories
		{Name: "Men's Clothing", Slug: "mens-clothing", ParentID: &fashion.ID, SortOrder: 1},
		{Name: "Women's Clothing", Slug: "womens-clothing", ParentID: &fashion.ID, SortOrder: 2},
		{Name: "Shoes", Slug: "shoes", ParentID: &fashion.ID, SortOrder: 3},
		{Name: "Accessories", Slug: "accessories", ParentID: &fashion.ID, SortOrder: 4},
		// Home subcategories
		{Name: "Furniture", Slug: "furniture", ParentID: &home.ID, SortOrder: 1},
		{Name: "Kitchen", Slug: "kitchen", ParentID: &home.ID, SortOrder: 2},
		{Name: "Bedding", Slug: "bedding", ParentID: &home.ID, SortOrder: 3},
		{Name: "Garden Tools", Slug: "garden-tools", ParentID: &home.ID, SortOrder: 4},
		// Sports subcategories
		{Name: "Fitness", Slug: "fitness", ParentID: &sports.ID, SortOrder: 1},
		{Name: "Cycling", Slug: "cycling", ParentID: &sports.ID, SortOrder: 2},
		{Name: "Camping", Slug: "camping", ParentID: &sports.ID, SortOrder: 3},
		// Books subcategories
		{Name: "Fiction", Slug: "fiction", ParentID: &books.ID, SortOrder: 1},
		{Name: "Non-Fiction", Slug: "non-fiction", ParentID: &books.ID, SortOrder: 2},
		{Name: "Textbooks", Slug: "textbooks", ParentID: &books.ID, SortOrder: 3},
	}

	for i := range subCategories {
		db.Create(&subCategories[i])
		categories = append(categories, subCategories[i])
	}

	log.Printf("   Created %d categories (8 parent + %d subcategories)", len(categories), len(subCategories))
	return categories
}

func createProducts(db *database.Database, categories []models.Category, users []models.User) []models.Product {
	// Get seller IDs
	sellers := []uuid.UUID{}
	for _, u := range users {
		if u.Role == "seller" {
			sellers = append(sellers, u.ID)
		}
	}

	products := []models.Product{}

	// Product data - mix of different price ranges and categories
	productData := []struct {
		Name           string
		Description    string
		Price          float64
		CompareAtPrice float64
		SKU            string
		CategorySlug   string
		Status         string
	}{
		// Electronics - High value items
		{"iPhone 15 Pro Max 256GB", "The most powerful iPhone ever with A17 Pro chip, titanium design, and 48MP camera system.", 1199.00, 1299.00, "IPHONE15PM-256", "smartphones", "active"},
		{"Samsung Galaxy S24 Ultra", "Premium Android flagship with S Pen, 200MP camera, and AI-powered features.", 1299.99, 1399.99, "SGS24U-256", "smartphones", "active"},
		{"Google Pixel 8 Pro", "Pure Android experience with exceptional camera capabilities and 7 years of updates.", 999.00, 0, "PIX8PRO-128", "smartphones", "active"},
		{"MacBook Pro 16\" M3 Max", "Professional laptop with M3 Max chip, stunning Liquid Retina XDR display.", 3499.00, 3699.00, "MBP16-M3MAX", "laptops", "active"},
		{"Dell XPS 15", "Premium Windows laptop with 4K OLED display and Intel Core i9.", 1899.99, 2099.99, "XPS15-I9", "laptops", "active"},
		{"ThinkPad X1 Carbon Gen 11", "Business ultrabook with legendary keyboard and all-day battery.", 1649.00, 0, "X1C-G11", "laptops", "active"},
		{"iPad Pro 12.9\" M2", "The ultimate iPad experience with M2 chip and mini-LED display.", 1099.00, 1199.00, "IPADPRO-129", "tablets", "active"},
		{"Sony WH-1000XM5", "Industry-leading noise cancellation and exceptional sound quality.", 349.99, 399.99, "WH1000XM5", "audio", "active"},
		{"AirPods Pro 2nd Gen", "Active Noise Cancellation, Transparency mode, and Adaptive Audio.", 249.00, 0, "APP2", "audio", "active"},
		{"Sony A7 IV", "Full-frame mirrorless camera for photo and video enthusiasts.", 2498.00, 2698.00, "A7IV-BODY", "cameras", "active"},
		{"PlayStation 5", "Next-gen gaming console with lightning-fast SSD and DualSense controller.", 499.99, 0, "PS5-DISC", "gaming", "active"},
		{"Xbox Series X", "The most powerful Xbox ever with 4K gaming at up to 120 FPS.", 499.99, 0, "XSX", "gaming", "active"},
		{"Nintendo Switch OLED", "Handheld gaming with vibrant 7-inch OLED screen.", 349.99, 0, "NSW-OLED", "gaming", "active"},

		// Fashion - Medium value items
		{"Classic Fit Oxford Shirt", "100% cotton button-down shirt, perfect for business casual.", 79.99, 99.99, "OXFORD-M-L", "mens-clothing", "active"},
		{"Slim Fit Chinos", "Comfortable stretch chinos with modern slim fit.", 59.99, 0, "CHINO-SLM-32", "mens-clothing", "active"},
		{"Premium Leather Belt", "Genuine leather belt with brushed steel buckle.", 49.99, 69.99, "BELT-BRN-34", "mens-clothing", "active"},
		{"Floral Summer Dress", "Lightweight floral print dress perfect for summer.", 89.99, 119.99, "DRESS-FLR-S", "womens-clothing", "active"},
		{"High-Waist Yoga Pants", "Buttery soft leggings with hidden pocket.", 49.99, 0, "YOGA-HW-M", "womens-clothing", "active"},
		{"Cashmere Cardigan", "Luxurious 100% cashmere cardigan in classic cut.", 199.99, 249.99, "CASH-CARD-M", "womens-clothing", "active"},
		{"Running Shoes Pro", "Lightweight performance running shoes with responsive cushioning.", 129.99, 159.99, "RUN-PRO-10", "shoes", "active"},
		{"Classic Leather Loafers", "Handcrafted Italian leather loafers.", 189.99, 0, "LOAF-BLK-9", "shoes", "active"},
		{"Designer Sunglasses", "UV protection with polarized lenses and titanium frame.", 299.99, 349.99, "SUN-POL-001", "accessories", "active"},
		{"Leather Crossbody Bag", "Premium leather crossbody with adjustable strap.", 149.99, 0, "BAG-CROSS-TAN", "accessories", "active"},

		// Home - Various price ranges
		{"Ergonomic Office Chair", "Full mesh office chair with lumbar support and adjustable armrests.", 399.99, 499.99, "CHAIR-ERG-BLK", "furniture", "active"},
		{"Standing Desk Electric", "Electric standing desk with memory presets, 60x30 inches.", 549.99, 649.99, "DESK-STAND-60", "furniture", "active"},
		{"Memory Foam Mattress Queen", "12-inch memory foam mattress with cooling gel layer.", 699.99, 899.99, "MATT-MF-Q", "bedding", "active"},
		{"Egyptian Cotton Sheet Set", "1000 thread count Egyptian cotton sheets.", 149.99, 199.99, "SHEET-EC-Q", "bedding", "active"},
		{"Professional Chef Knife Set", "8-piece German steel knife set with block.", 249.99, 329.99, "KNIFE-SET-8", "kitchen", "active"},
		{"Cast Iron Dutch Oven", "6-quart enameled cast iron Dutch oven.", 129.99, 0, "DUTCH-6QT", "kitchen", "active"},
		{"Cordless Lawn Mower", "40V lithium-ion cordless lawn mower, 21-inch cut.", 449.99, 529.99, "MOWER-40V", "garden-tools", "active"},
		{"Garden Tool Set 12-Piece", "Complete garden tool set with carrying bag.", 79.99, 99.99, "GARDEN-12PC", "garden-tools", "active"},

		// Sports - Various items
		{"Adjustable Dumbbell Set", "Adjustable dumbbells 5-52.5 lbs with stand.", 349.99, 449.99, "DUMB-ADJ-52", "fitness", "active"},
		{"Yoga Mat Premium", "Extra thick non-slip yoga mat with carrying strap.", 39.99, 0, "YOGA-MAT-PRO", "fitness", "active"},
		{"Resistance Band Set", "5 levels of resistance bands with handles and door anchor.", 29.99, 39.99, "BAND-SET-5", "fitness", "active"},
		{"Carbon Fiber Road Bike", "Professional grade carbon fiber road bike.", 2499.99, 2999.99, "BIKE-CF-ROAD", "cycling", "active"},
		{"Bike Helmet Pro", "Aerodynamic cycling helmet with MIPS protection.", 149.99, 0, "HELM-BIKE-M", "cycling", "active"},
		{"4-Person Tent", "Waterproof 4-season tent with easy setup.", 299.99, 399.99, "TENT-4P", "camping", "active"},
		{"Sleeping Bag 0°F", "Mummy sleeping bag rated to 0°F.", 129.99, 159.99, "SLEEP-0F", "camping", "active"},
		{"Camping Cookware Set", "Lightweight aluminum cookware set for camping.", 49.99, 0, "CAMP-COOK", "camping", "active"},

		// Books - Low to medium value
		{"The Great Gatsby", "F. Scott Fitzgerald's American classic. Hardcover edition.", 14.99, 0, "BOOK-GATSBY", "fiction", "active"},
		{"1984 by George Orwell", "Dystopian masterpiece. Anniversary edition.", 12.99, 15.99, "BOOK-1984", "fiction", "active"},
		{"Harry Potter Complete Set", "All 7 Harry Potter books in collector's box set.", 149.99, 199.99, "HP-BOX-SET", "fiction", "active"},
		{"Atomic Habits", "James Clear's bestseller on building better habits.", 16.99, 0, "BOOK-ATOMIC", "non-fiction", "active"},
		{"Sapiens", "Yuval Noah Harari's brief history of humankind.", 18.99, 24.99, "BOOK-SAPIENS", "non-fiction", "active"},
		{"Introduction to Algorithms", "The definitive computer science textbook.", 89.99, 0, "TEXT-ALGO", "textbooks", "active"},

		// Beauty & Health
		{"Vitamin C Serum", "20% Vitamin C serum with hyaluronic acid.", 29.99, 39.99, "SERUM-VIT-C", "beauty-health", "active"},
		{"Retinol Night Cream", "Anti-aging night cream with retinol.", 44.99, 59.99, "CREAM-RETINOL", "beauty-health", "active"},
		{"Electric Toothbrush Pro", "Sonic toothbrush with 5 modes and smart timer.", 79.99, 99.99, "TOOTH-SONIC", "beauty-health", "active"},

		// Toys & Games
		{"LEGO Star Wars Millennium Falcon", "Ultimate Collector's Edition with 7541 pieces.", 849.99, 0, "LEGO-MF-UCS", "toys-games", "active"},
		{"Board Game Collection", "Classic board games: Monopoly, Scrabble, Risk.", 49.99, 69.99, "BOARD-CLASSIC", "toys-games", "active"},

		// Automotive
		{"Dash Cam 4K", "4K front and rear dash cam with night vision.", 149.99, 199.99, "DASH-4K-DR", "automotive", "active"},
		{"Portable Jump Starter", "2000A peak jump starter with USB-C power bank.", 99.99, 129.99, "JUMP-2000A", "automotive", "active"},

		// Some draft/inactive products
		{"Unreleased Smartphone", "Next generation smartphone - coming soon.", 999.99, 0, "PHONE-SOON", "smartphones", "draft"},
		{"Discontinued Headphones", "Previous model - being phased out.", 199.99, 299.99, "HEAD-OLD", "audio", "inactive"},
	}

	// Create category map for quick lookup
	categoryMap := make(map[string]uuid.UUID)
	for _, cat := range categories {
		categoryMap[cat.Slug] = cat.ID
	}

	for i, p := range productData {
		catID, exists := categoryMap[p.CategorySlug]
		if !exists {
			// Use parent category
			for _, cat := range categories {
				if cat.ParentID == nil {
					catID = cat.ID
					break
				}
			}
		}

		imagesJSON, _ := json.Marshal([]string{
			fmt.Sprintf("https://images.example.com/products/%s-1.jpg", p.SKU),
			fmt.Sprintf("https://images.example.com/products/%s-2.jpg", p.SKU),
		})

		product := models.Product{
			Name:           p.Name,
			Slug:           fmt.Sprintf("%s-%s", generateSlug(p.Name), p.SKU),
			Description:    p.Description,
			Price:          p.Price,
			CompareAtPrice: p.CompareAtPrice,
			SKU:            p.SKU,
			CategoryID:     catID,
			SellerID:       sellers[i%len(sellers)],
			Images:         string(imagesJSON),
			Attributes:     "{}", // Valid empty JSON object
			Status:         p.Status,
		}

		if err := db.Create(&product).Error; err != nil {
			log.Printf("   Error creating product %s: %v", p.Name, err)
			continue
		}
		products = append(products, product)
	}

	log.Printf("   Created %d products", len(products))
	return products
}

func createAddresses(db *database.Database, users []models.User) {
	addresses := []struct {
		UserIndex    int
		Type         string
		FirstName    string
		LastName     string
		AddressLine1 string
		City         string
		State        string
		PostalCode   string
		Country      string
		IsDefault    bool
	}{
		// John Doe - 2 addresses
		{6, "shipping", "John", "Doe", "123 Main Street", "New York", "NY", "10001", "USA", true},
		{6, "billing", "John", "Doe", "456 Work Ave", "New York", "NY", "10002", "USA", false},
		// Jane Smith
		{7, "both", "Jane", "Smith", "789 Oak Lane", "Los Angeles", "CA", "90001", "USA", true},
		// Mike Johnson
		{8, "shipping", "Mike", "Johnson", "321 Pine Road", "Chicago", "IL", "60601", "USA", true},
		// Sarah Williams - multiple addresses
		{9, "shipping", "Sarah", "Williams", "555 Maple Drive", "Houston", "TX", "77001", "USA", true},
		{9, "shipping", "Sarah", "Williams", "777 Office Park", "Houston", "TX", "77002", "USA", false},
		{9, "billing", "Sarah", "Williams", "888 Billing Center", "Houston", "TX", "77003", "USA", false},
		// David Brown
		{10, "both", "David", "Brown", "999 Cedar Boulevard", "Phoenix", "AZ", "85001", "USA", true},
		// More customers...
		{11, "shipping", "Emily", "Davis", "111 First Street", "Philadelphia", "PA", "19101", "USA", true},
		{12, "both", "Chris", "Miller", "222 Second Avenue", "San Antonio", "TX", "78201", "USA", true},
		{13, "shipping", "Lisa", "Wilson", "333 Third Lane", "San Diego", "CA", "92101", "USA", true},
		{14, "both", "Alex", "Moore", "444 Fourth Road", "Dallas", "TX", "75201", "USA", true},
		{15, "shipping", "Rachel", "Taylor", "555 Fifth Drive", "San Jose", "CA", "95101", "USA", true},
	}

	count := 0
	for _, a := range addresses {
		if a.UserIndex >= len(users) {
			continue
		}
		address := models.Address{
			UserID:       users[a.UserIndex].ID,
			Type:         a.Type,
			FirstName:    a.FirstName,
			LastName:     a.LastName,
			AddressLine1: a.AddressLine1,
			City:         a.City,
			State:        a.State,
			PostalCode:   a.PostalCode,
			Country:      a.Country,
			IsDefault:    a.IsDefault,
		}
		db.Create(&address)
		count++
	}

	log.Printf("   Created %d addresses", count)
}

func createInventory(db *database.Database, products []models.Product) {
	warehouses := []struct {
		ID   uuid.UUID
		Name string
	}{
		{uuid.New(), "East Coast Warehouse"},
		{uuid.New(), "West Coast Warehouse"},
		{uuid.New(), "Central Distribution"},
	}

	count := 0
	for _, product := range products {
		if product.Status != "active" {
			continue
		}

		// Add inventory in 1-3 warehouses per product
		numWarehouses := rand.Intn(3) + 1
		for i := 0; i < numWarehouses; i++ {
			warehouse := warehouses[i]

			// Varying stock levels
			quantity := rand.Intn(500) + 10
			reserved := rand.Intn(quantity / 4)

			inventory := models.Inventory{
				ProductID:       product.ID,
				WarehouseID:     warehouse.ID,
				WarehouseName:   warehouse.Name,
				Quantity:        quantity,
				ReservedQty:     reserved,
				ReorderLevel:    10,
				ReorderQuantity: 50,
			}
			db.Create(&inventory)
			count++
		}
	}

	log.Printf("   Created %d inventory records across 3 warehouses", count)
}

func createCoupons(db *database.Database) []models.Coupon {
	now := time.Now()
	future := now.AddDate(0, 6, 0) // 6 months from now
	past := now.AddDate(0, -1, 0)  // 1 month ago

	coupons := []models.Coupon{
		// Active coupons
		{Code: "WELCOME10", Description: "10% off for new customers", Type: "percentage", Value: 10, MinOrderAmount: 50, UsageLimit: 1000, PerUserLimit: 1, StartDate: &now, EndDate: &future, IsActive: true, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		{Code: "SAVE20", Description: "$20 off orders over $100", Type: "fixed", Value: 20, MinOrderAmount: 100, UsageLimit: 500, PerUserLimit: 2, StartDate: &now, EndDate: &future, IsActive: true, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		{Code: "FREESHIP", Description: "Free shipping on all orders", Type: "free_shipping", Value: 9.99, MinOrderAmount: 25, IsActive: true, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		{Code: "SUMMER25", Description: "25% off summer sale", Type: "percentage", Value: 25, MaxDiscount: 100, MinOrderAmount: 75, StartDate: &now, EndDate: &future, IsActive: true, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		{Code: "VIP50", Description: "50% off VIP exclusive", Type: "percentage", Value: 50, MaxDiscount: 200, MinOrderAmount: 200, UsageLimit: 100, PerUserLimit: 1, StartDate: &now, EndDate: &future, IsActive: true, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		{Code: "FLASH15", Description: "15% flash sale", Type: "percentage", Value: 15, UsageLimit: 200, StartDate: &now, EndDate: &future, IsActive: true, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		// Expired coupon
		{Code: "EXPIRED20", Description: "Old promotion - expired", Type: "percentage", Value: 20, StartDate: &past, EndDate: &now, IsActive: false, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		// High-usage coupon
		{Code: "POPULAR10", Description: "Popular 10% discount", Type: "percentage", Value: 10, UsageLimit: 1000, UsageCount: 847, StartDate: &now, EndDate: &future, IsActive: true, ApplicableCategories: "[]", ApplicableProducts: "[]"},
	}

	for i := range coupons {
		if err := db.Create(&coupons[i]).Error; err != nil {
			log.Printf("   Error creating coupon %s: %v", coupons[i].Code, err)
		}
	}

	log.Printf("   Created %d coupons", len(coupons))
	return coupons
}

func createCartsWithItems(db *database.Database, users []models.User, products []models.Product) {
	// Get active products only
	activeProducts := []models.Product{}
	for _, p := range products {
		if p.Status == "active" {
			activeProducts = append(activeProducts, p)
		}
	}

	// Add items to some users' carts
	cartsWithItems := 0
	for i, user := range users {
		if user.Role != "customer" || i%3 != 0 {
			continue
		}

		// Find user's cart
		var cart models.Cart
		if err := db.Where("user_id = ?", user.ID).First(&cart).Error; err != nil {
			continue
		}

		// Add 1-5 random products
		numItems := rand.Intn(5) + 1
		for j := 0; j < numItems; j++ {
			product := activeProducts[rand.Intn(len(activeProducts))]

			// Check if already in cart
			var existing models.CartItem
			if db.Where("cart_id = ? AND product_id = ?", cart.ID, product.ID).First(&existing).Error == nil {
				continue
			}

			cartItem := models.CartItem{
				CartID:    cart.ID,
				ProductID: product.ID,
				Quantity:  rand.Intn(3) + 1,
			}
			db.Create(&cartItem)
		}
		cartsWithItems++
	}

	log.Printf("   Added items to %d carts", cartsWithItems)
}

func createOrders(db *database.Database, users []models.User, products []models.Product, coupons []models.Coupon) {
	activeProducts := []models.Product{}
	for _, p := range products {
		if p.Status == "active" {
			activeProducts = append(activeProducts, p)
		}
	}

	// Get customers with addresses
	customerIDs := []uuid.UUID{}
	for _, u := range users {
		if u.Role == "customer" && u.Status == "active" {
			customerIDs = append(customerIDs, u.ID)
		}
	}

	statuses := []string{"pending", "confirmed", "processing", "shipped", "delivered", "cancelled", "refunded"}
	paymentMethods := []string{"card", "bank_transfer", "wallet", "cod"}
	paymentStatuses := []string{"pending", "completed", "failed", "refunded"}

	ordersCreated := 0

	// Create varying number of orders per customer
	for _, userID := range customerIDs {
		// Find user's address
		var address models.Address
		if err := db.Where("user_id = ?", userID).First(&address).Error; err != nil {
			continue
		}

		// 0-10 orders per customer (varying activity levels)
		numOrders := rand.Intn(11)
		for i := 0; i < numOrders; i++ {
			// Random number of items (1-6)
			numItems := rand.Intn(6) + 1
			var subtotal float64 = 0
			orderItems := []struct {
				Product  models.Product
				Quantity int
			}{}

			// Select random products
			for j := 0; j < numItems; j++ {
				product := activeProducts[rand.Intn(len(activeProducts))]
				quantity := rand.Intn(3) + 1
				subtotal += product.Price * float64(quantity)
				orderItems = append(orderItems, struct {
					Product  models.Product
					Quantity int
				}{product, quantity})
			}

			// Apply discount randomly
			var discount float64 = 0
			var couponID *uuid.UUID = nil
			if rand.Float32() < 0.3 && len(coupons) > 0 {
				coupon := coupons[rand.Intn(len(coupons))]
				if coupon.IsActive {
					if coupon.Type == "percentage" {
						discount = subtotal * (coupon.Value / 100)
						if coupon.MaxDiscount > 0 && discount > coupon.MaxDiscount {
							discount = coupon.MaxDiscount
						}
					} else if coupon.Type == "fixed" {
						discount = coupon.Value
					}
					couponID = &coupon.ID
				}
			}

			shipping := 9.99
			if subtotal > 100 {
				shipping = 0
			}
			tax := (subtotal - discount) * 0.08
			total := subtotal - discount + shipping + tax

			status := statuses[rand.Intn(len(statuses))]
			orderNumber := fmt.Sprintf("ORD-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8])

			order := models.Order{
				OrderNumber:       orderNumber,
				UserID:            userID,
				Status:            status,
				Subtotal:          subtotal,
				Discount:          discount,
				ShippingCost:      shipping,
				Tax:               tax,
				Total:             total,
				CouponID:          couponID,
				ShippingAddressID: address.ID,
			}

			if err := db.Create(&order).Error; err != nil {
				continue
			}

			// Create order items
			for _, item := range orderItems {
				orderItem := models.OrderItem{
					OrderID:     order.ID,
					ProductID:   item.Product.ID,
					ProductName: item.Product.Name,
					SKU:         item.Product.SKU,
					Quantity:    item.Quantity,
					UnitPrice:   item.Product.Price,
					TotalPrice:  item.Product.Price * float64(item.Quantity),
				}
				db.Create(&orderItem)
			}

			// Create payment
			paymentStatus := "pending"
			if status == "delivered" || status == "shipped" {
				paymentStatus = "completed"
			} else if status == "cancelled" {
				paymentStatus = "failed"
			} else if status == "refunded" {
				paymentStatus = "refunded"
			}
			if paymentStatus == "pending" && rand.Float32() < 0.5 {
				paymentStatus = paymentStatuses[rand.Intn(len(paymentStatuses))]
			}

			payment := models.Payment{
				OrderID:         order.ID,
				Amount:          total,
				Currency:        "USD",
				Method:          paymentMethods[rand.Intn(len(paymentMethods))],
				Status:          paymentStatus,
				GatewayResponse: "{}",
			}
			db.Create(&payment)

			ordersCreated++
		}
	}

	log.Printf("   Created %d orders with items and payments", ordersCreated)
}

func createReviews(db *database.Database, users []models.User, products []models.Product) {
	activeProducts := []models.Product{}
	for _, p := range products {
		if p.Status == "active" {
			activeProducts = append(activeProducts, p)
		}
	}

	if len(activeProducts) == 0 {
		log.Printf("   No active products to review")
		return
	}

	customers := []models.User{}
	for _, u := range users {
		if u.Role == "customer" && u.Status == "active" {
			customers = append(customers, u)
		}
	}

	if len(customers) == 0 {
		log.Printf("   No customers to create reviews")
		return
	}

	titles := []string{
		"Great product!", "Exceeded expectations", "Good value", "Highly recommend",
		"Excellent quality", "Perfect fit", "Love it!", "Just okay",
		"Not as expected", "Would buy again", "Five stars!", "Solid purchase",
	}

	contents := []string{
		"This product is exactly what I was looking for. Fast shipping and great quality!",
		"I've been using this for a few weeks now and it's holding up well. Very satisfied.",
		"Good product for the price. Some minor issues but overall happy with my purchase.",
		"Exceeded my expectations! The quality is top-notch and customer service was excellent.",
		"It's okay. Does what it's supposed to but nothing special.",
		"Amazing! I've already recommended it to all my friends and family.",
		"The product arrived quickly and was well packaged. Works as described.",
		"Great value for money. Would definitely purchase again.",
		"Decent quality but took longer to arrive than expected.",
		"Absolutely love this product! Five stars all the way.",
	}

	reviewsCreated := 0

	// Create reviews for products (not all products will have reviews)
	for _, product := range activeProducts {
		// 0-8 reviews per product
		numReviews := rand.Intn(9)
		usedCustomers := make(map[uuid.UUID]bool)

		for i := 0; i < numReviews; i++ {
			// Select random customer who hasn't reviewed yet
			customer := customers[rand.Intn(len(customers))]
			if usedCustomers[customer.ID] {
				continue
			}
			usedCustomers[customer.ID] = true

			// Rating weighted towards higher values
			rating := rand.Intn(5) + 1
			if rand.Float32() < 0.6 {
				rating = rand.Intn(2) + 4 // 4 or 5
			}

			review := models.Review{
				ProductID:        product.ID,
				UserID:           customer.ID,
				Rating:           rating,
				Title:            titles[rand.Intn(len(titles))],
				Content:          contents[rand.Intn(len(contents))],
				Images:           "[]", // Valid empty JSON array
				VerifiedPurchase: rand.Float32() < 0.7,
				HelpfulCount:     rand.Intn(50),
			}
			if err := db.Create(&review).Error; err == nil {
				reviewsCreated++
			}
		}
	}

	log.Printf("   Created %d reviews", reviewsCreated)
}

func generateSlug(name string) string {
	slug := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			slug += string(c)
		} else if c >= 'A' && c <= 'Z' {
			slug += string(c + 32)
		} else if c == ' ' {
			slug += "-"
		}
	}
	return slug
}

func printSummary(db *database.Database) {
	log.Println("\n📊 Data Summary:")
	log.Println("═══════════════════════════════════════")

	tables := []string{"users", "categories", "products", "addresses", "inventories", "coupons", "carts", "cart_items", "orders", "order_items", "payments", "reviews"}

	for _, table := range tables {
		var count int64
		db.Table(table).Count(&count)
		log.Printf("   %-15s: %d records", table, count)
	}

	log.Println("═══════════════════════════════════════")
	log.Println("\n🔑 Test Credentials:")
	log.Println("   Admin:    admin@marketplace.com / Password123!")
	log.Println("   Seller:   seller1@marketplace.com / Password123!")
	log.Println("   Customer: john.doe@email.com / Password123!")
	log.Println("")
}
