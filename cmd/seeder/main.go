package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marketplace-api/internal/auth"
	"github.com/marketplace-api/internal/config"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/models"
)

// Configuration for FAST seeding (~100MB stored, but pulls 300-400MB due to joins/relations)
const (
	// Reduced counts for faster seeding
	NUM_USERS           = 100  // 100 users
	NUM_SELLERS         = 10   // 10 sellers
	NUM_CATEGORIES      = 30   // 30 categories
	NUM_PRODUCTS        = 500  // 500 products (but with HUGE descriptions)
	NUM_ORDERS          = 2000 // 2000 orders
	REVIEWS_PER_PRODUCT = 10   // ~10 reviews per product = 5,000 reviews

	// LARGER text fields to compensate for fewer records
	PRODUCT_DESC_SIZE   = 25000 // ~25KB description per product (was 8KB)
	PRODUCT_ATTRS_SIZE  = 15000 // ~15KB JSON attributes per product (was 5KB)
	REVIEW_CONTENT_SIZE = 8000  // ~8KB review content (was 2KB)
	ORDER_NOTES_SIZE    = 3000  // ~3KB order notes (was 1KB)
	CATEGORY_DESC_SIZE  = 2000  // ~2KB category description

	// Batch sizes for FAST inserts
	BATCH_SIZE = 50
)

// Helper data for generating random content
var (
	firstNames = []string{"John", "Jane", "Michael", "Sarah", "David", "Emily", "Chris", "Lisa", "Alex", "Rachel",
		"James", "Olivia", "Noah", "Sophia", "Liam", "Ava", "William", "Mia", "Ethan", "Isabella"}
	lastNames = []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
		"Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin"}
	materials = []string{"Premium Aluminum", "Stainless Steel", "Carbon Fiber", "Genuine Leather", "Organic Cotton",
		"Recycled Plastic", "Tempered Glass", "Solid Oak", "Bamboo Fiber", "Aerospace Titanium"}
	colors = []string{"Midnight Black", "Arctic White", "Space Gray", "Rose Gold", "Navy Blue",
		"Forest Green", "Burgundy Red", "Champagne", "Graphite", "Pearl White"}
	countries    = []string{"USA", "Germany", "Japan", "South Korea", "Taiwan", "China", "Vietnam", "Mexico"}
	productTypes = []string{"Electronics", "Fashion", "Home & Garden", "Sports", "Books", "Beauty", "Toys", "Automotive"}
)

func main() {
	log.Println("🌱 Starting FAST data seeder (Target: ~100MB stored, 300-400MB on query)...")
	log.Printf("📊 Configuration:")
	log.Printf("   - Products: %d (each with ~%dKB description, ~%dKB attributes)", NUM_PRODUCTS, PRODUCT_DESC_SIZE/1000, PRODUCT_ATTRS_SIZE/1000)
	log.Printf("   - Reviews: ~%d (each with ~%dKB content)", NUM_PRODUCTS*REVIEWS_PER_PRODUCT, REVIEW_CONTENT_SIZE/1000)
	log.Printf("   - Orders: %d (each with ~%dKB notes)", NUM_ORDERS, ORDER_NOTES_SIZE/1000)
	log.Printf("   - Estimated stored data: ~%.0fMB", estimateDataSize())
	log.Printf("   - Estimated seeding time: 3-5 minutes")

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
	log.Println("📊 Seeding dataset...")
	startTime := time.Now()
	seedData(db)
	elapsed := time.Since(startTime)

	log.Printf("✅ Data seeding completed in %v!", elapsed)
}

func estimateDataSize() float64 {
	productData := float64(NUM_PRODUCTS) * float64(PRODUCT_DESC_SIZE+PRODUCT_ATTRS_SIZE+500) / 1024 / 1024
	reviewData := float64(NUM_PRODUCTS*REVIEWS_PER_PRODUCT) * float64(REVIEW_CONTENT_SIZE+200) / 1024 / 1024
	orderData := float64(NUM_ORDERS) * float64(ORDER_NOTES_SIZE+500) / 1024 / 1024
	categoryData := float64(NUM_CATEGORIES) * float64(CATEGORY_DESC_SIZE+200) / 1024 / 1024
	return productData + reviewData + orderData + categoryData
}

// ==================== Helper Functions ====================

func generateRandomMaterial() string { return materials[rand.Intn(len(materials))] }
func generateRandomColor() string    { return colors[rand.Intn(len(colors))] }
func generateRandomCountry() string  { return countries[rand.Intn(len(countries))] }

func generateRandomFeatures(count int) []string {
	features := []string{
		"Advanced Temperature Control", "Intelligent Power Management", "Multi-layer Protection",
		"Smart Connectivity", "Enhanced Durability", "Eco-Friendly Materials", "Precision Engineering",
		"Ergonomic Design", "Quick-Release Mechanism", "Universal Compatibility", "Extended Battery",
		"High-Resolution Display", "Noise Cancellation", "Water Resistant", "Shock Absorption",
	}
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = features[rand.Intn(len(features))]
	}
	return result
}

func generateTechnicalSpecs() map[string]interface{} {
	return map[string]interface{}{
		"processor": map[string]interface{}{
			"type":  fmt.Sprintf("ARM Cortex-A%d", 70+rand.Intn(10)),
			"cores": 2 << rand.Intn(4), "clockSpeed": fmt.Sprintf("%.1f GHz", 1.0+rand.Float64()*3.0),
		},
		"memory":  map[string]interface{}{"type": "DDR5", "size": fmt.Sprintf("%d GB", 4<<rand.Intn(4))},
		"storage": map[string]interface{}{"type": "NVMe SSD", "capacity": fmt.Sprintf("%d GB", 128<<rand.Intn(4))},
		"display": map[string]interface{}{"size": fmt.Sprintf("%.1f inches", 5.0+rand.Float64()*10.0)},
		"battery": map[string]interface{}{"capacity": fmt.Sprintf("%d mAh", 3000+rand.Intn(7000))},
		"sensors": []string{"Accelerometer", "Gyroscope", "Proximity", "Light", "Barometer"},
	}
}

// generateLargeText generates large text of specified size
func generateLargeText(size int, topic string) string {
	paragraphs := []string{
		"This premium product represents the pinnacle of modern engineering and design excellence. Crafted with meticulous attention to detail, it combines cutting-edge technology with timeless aesthetics. Our team of experts has spent countless hours perfecting every aspect to ensure it meets the highest standards of quality and performance. The innovative features provide an exceptional user experience that sets new benchmarks in the industry.",
		"Experience unparalleled craftsmanship with materials sourced from the finest suppliers around the globe. Each component undergoes rigorous quality control testing to guarantee durability and reliability. The design ensures optimal functionality while maintaining an elegant appearance that complements any setting. We believe in delivering products that not only meet but exceed customer expectations.",
		"Built to exceed expectations, this product incorporates the latest advancements in technology and manufacturing. The state-of-the-art features provide an exceptional user experience, whether you're a first-time buyer or a seasoned professional. We stand behind our commitment to excellence with comprehensive warranty coverage and dedicated customer support available around the clock.",
		"Sustainability meets innovation in this remarkable product. We've implemented eco-friendly manufacturing practices without compromising on performance or quality. The recyclable packaging and energy-efficient design reflect our dedication to environmental responsibility. Join us in creating a better future for generations to come while enjoying premium quality.",
		"Join millions of satisfied customers who have discovered the difference that true quality makes. Our products have received numerous industry awards and certifications, recognizing our unwavering commitment to innovation and customer satisfaction. From concept to delivery, every step of our process is designed to delight and inspire confidence.",
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[%s Product Details]\n\n", topic))
	for builder.Len() < size {
		builder.WriteString(paragraphs[rand.Intn(len(paragraphs))])
		builder.WriteString("\n\n")
	}
	result := builder.String()
	if len(result) > size {
		result = result[:size]
	}
	return result
}

// generateLargeProductAttributes generates large JSON attributes
func generateLargeProductAttributes(productType string) string {
	attrs := map[string]interface{}{
		"productType":    productType,
		"brand":          fmt.Sprintf("Brand-%d", rand.Intn(100)),
		"model":          fmt.Sprintf("Model-%d-%d", rand.Intn(1000), rand.Intn(1000)),
		"specifications": generateTechnicalSpecs(),
		"features":       generateRandomFeatures(25),
		"materials":      []string{generateRandomMaterial(), generateRandomMaterial(), generateRandomMaterial()},
		"colors":         []string{generateRandomColor(), generateRandomColor(), generateRandomColor()},
		"certifications": []string{"CE", "FCC", "RoHS", "ISO 9001", "ISO 14001", "UL Listed", "Energy Star"},
		"warranty":       map[string]string{"standard": "2 years", "extended": "5 years available"},
		"manufacturing":  map[string]string{"origin": generateRandomCountry(), "assembly": generateRandomCountry()},
	}

	// Add padding data to reach target size
	paddingNeeded := PRODUCT_ATTRS_SIZE - 2000
	if paddingNeeded > 0 {
		extraData := make(map[string]string)
		for i := 0; paddingNeeded > 0; i++ {
			key := fmt.Sprintf("spec_%d", i)
			value := strings.Repeat(fmt.Sprintf("data_%d_value_", i), 20)
			extraData[key] = value
			paddingNeeded -= len(key) + len(value) + 10
		}
		attrs["extendedSpecs"] = extraData
	}

	jsonBytes, _ := json.Marshal(attrs)
	return string(jsonBytes)
}

func generateLargeReviewContent() string {
	templates := []string{
		"I've been using this product for several weeks now and I have to say, it has completely exceeded my expectations. The build quality is exceptional, and every feature works exactly as advertised. The packaging was impressive - you can tell this company cares about the unboxing experience. I've recommended it to all my friends and family members who are looking for similar products.",
		"After extensive research and comparing multiple options on the market, I decided to go with this product. It was the best decision I've made in a long time! The performance is outstanding, and the attention to detail is remarkable. Customer service was also incredibly helpful when I had questions about setup and configuration.",
		"This is my third purchase from this brand, and they consistently deliver quality that exceeds expectations. The quality consistency across their product line is impressive. This particular item fits perfectly into my daily workflow and has significantly increased my productivity. The learning curve was minimal thanks to the intuitive design.",
		"I was initially skeptical given the price point compared to competitors, but this product has proven its value time and time again. It's been through some tough situations and still performs flawlessly. The durability is exceptional, and I appreciate all the thoughtful design choices throughout.",
	}

	var builder strings.Builder
	for builder.Len() < REVIEW_CONTENT_SIZE {
		builder.WriteString(templates[rand.Intn(len(templates))])
		builder.WriteString(" ")
	}
	result := builder.String()
	if len(result) > REVIEW_CONTENT_SIZE {
		result = result[:REVIEW_CONTENT_SIZE]
	}
	return result
}

func generateLargeOrderNotes() string {
	notes := []string{
		"Please handle with care - fragile items inside. Gift wrapping requested.",
		"Leave package at back door if no one home. Call before delivery.",
		"This is a replacement order for damaged item. Priority shipping requested.",
		"Include invoice with package. Deliver to office address during weekday hours.",
		"Special packaging required for international destination. Signature required.",
	}
	var builder strings.Builder
	for builder.Len() < ORDER_NOTES_SIZE {
		builder.WriteString(notes[rand.Intn(len(notes))])
		builder.WriteString(fmt.Sprintf(" Reference: REF-%d-%d. ", rand.Intn(100000), rand.Intn(1000)))
	}
	result := builder.String()
	if len(result) > ORDER_NOTES_SIZE {
		result = result[:ORDER_NOTES_SIZE]
	}
	return result
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

// ==================== Database Operations ====================

func flushData(db *database.Database) {
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	tables := []string{"refresh_tokens", "cart_items", "carts", "order_items", "payments", "orders",
		"reviews", "inventories", "products", "categories", "addresses", "coupons", "users"}
	for _, table := range tables {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
		log.Printf("   Cleared table: %s", table)
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

func seedData(db *database.Database) {
	log.Println("👥 Creating users...")
	users := createUsers(db)

	log.Println("📁 Creating categories...")
	categories := createCategories(db)

	log.Println("🛍️ Creating products (with large descriptions)...")
	products := createProducts(db, categories, users)

	log.Println("📍 Creating addresses...")
	createAddresses(db, users)

	log.Println("📦 Creating inventory...")
	createInventory(db, products)

	log.Println("🎟️ Creating coupons...")
	coupons := createCoupons(db)

	log.Println("🛒 Creating carts...")
	createCartsWithItems(db, users, products)

	log.Println("📋 Creating orders (with large notes)...")
	createOrders(db, users, products, coupons)

	log.Println("⭐ Creating reviews (with large content)...")
	createReviews(db, users, products)

	printSummary(db)
}

func createUsers(db *database.Database) []models.User {
	hashedPassword, _ := auth.HashPassword("Password123!")

	users := []models.User{
		{Email: "admin@marketplace.com", Password: hashedPassword, FirstName: "Admin", LastName: "User", Role: "admin", Status: "active", Phone: "+1-555-0100"},
		{Email: "seller1@marketplace.com", Password: hashedPassword, FirstName: "Tech", LastName: "Store", Role: "seller", Status: "active", Phone: "+1-555-0101"},
		{Email: "seller2@marketplace.com", Password: hashedPassword, FirstName: "Fashion", LastName: "Hub", Role: "seller", Status: "active", Phone: "+1-555-0102"},
		{Email: "seller3@marketplace.com", Password: hashedPassword, FirstName: "Home", LastName: "Essentials", Role: "seller", Status: "active", Phone: "+1-555-0103"},
		{Email: "john.doe@email.com", Password: hashedPassword, FirstName: "John", LastName: "Doe", Role: "customer", Status: "active", Phone: "+1-555-1001", EmailVerified: true},
	}

	for i := range users {
		db.Create(&users[i])
		db.Create(&models.Cart{UserID: users[i].ID})
	}

	// Add more sellers
	for i := 4; i <= NUM_SELLERS; i++ {
		user := models.User{
			Email: fmt.Sprintf("seller%d@marketplace.com", i), Password: hashedPassword,
			FirstName: firstNames[rand.Intn(len(firstNames))], LastName: lastNames[rand.Intn(len(lastNames))],
			Role: "seller", Status: "active", Phone: fmt.Sprintf("+1-555-%04d", 200+i),
		}
		db.Create(&user)
		db.Create(&models.Cart{UserID: user.ID})
		users = append(users, user)
	}

	// Add customers
	for i := 0; i < NUM_USERS-len(users); i++ {
		user := models.User{
			Email: fmt.Sprintf("customer%d@email.com", i), Password: hashedPassword,
			FirstName: firstNames[rand.Intn(len(firstNames))], LastName: lastNames[rand.Intn(len(lastNames))],
			Role: "customer", Status: "active", Phone: fmt.Sprintf("+1-555-%04d", 2000+i), EmailVerified: rand.Float32() < 0.7,
		}
		db.Create(&user)
		db.Create(&models.Cart{UserID: user.ID})
		users = append(users, user)
	}

	log.Printf("   Created %d users", len(users))
	return users
}

func createCategories(db *database.Database) []models.Category {
	categories := []models.Category{}

	parentCats := []string{"Electronics", "Fashion", "Home & Garden", "Sports", "Books", "Beauty", "Toys", "Automotive"}
	for i, name := range parentCats {
		cat := models.Category{
			Name: name, Slug: generateSlug(name), Description: generateLargeText(CATEGORY_DESC_SIZE, name), SortOrder: i + 1,
		}
		db.Create(&cat)
		categories = append(categories, cat)
	}

	subCats := []struct{ Name, Parent string }{
		{"Smartphones", "electronics"}, {"Laptops", "electronics"}, {"Audio", "electronics"},
		{"Men's Clothing", "fashion"}, {"Women's Clothing", "fashion"}, {"Shoes", "fashion"},
		{"Furniture", "furniture"}, {"Kitchen", "home-garden"}, {"Bedding", "home-garden"},
		{"Fitness", "sports"}, {"Cycling", "sports"}, {"Camping", "sports"},
		{"Fiction", "books"}, {"Non-Fiction", "books"}, {"Skincare", "beauty"},
		{"Board Games", "toys"}, {"Car Electronics", "automotive"},
	}

	parentMap := make(map[string]uuid.UUID)
	for _, cat := range categories {
		parentMap[cat.Slug] = cat.ID
	}

	for i, sc := range subCats {
		parentID := parentMap[sc.Parent]
		cat := models.Category{
			Name: sc.Name, Slug: generateSlug(sc.Name), Description: generateLargeText(CATEGORY_DESC_SIZE, sc.Name),
			ParentID: &parentID, SortOrder: i + 1,
		}
		db.Create(&cat)
		categories = append(categories, cat)
	}

	log.Printf("   Created %d categories", len(categories))
	return categories
}

func createProducts(db *database.Database, categories []models.Category, users []models.User) []models.Product {
	sellers := []uuid.UUID{}
	for _, u := range users {
		if u.Role == "seller" {
			sellers = append(sellers, u.ID)
		}
	}

	catIDs := []uuid.UUID{}
	for _, cat := range categories {
		catIDs = append(catIDs, cat.ID)
	}

	products := []models.Product{}
	productNames := []string{"Premium Pro", "Ultra Edition", "Classic Series", "Elite Model", "Deluxe Package",
		"Professional Grade", "Smart Connected", "Wireless Freedom", "High Performance", "Limited Edition"}

	for i := 0; i < NUM_PRODUCTS; i++ {
		productType := productTypes[rand.Intn(len(productTypes))]
		name := fmt.Sprintf("%s %s %d", productType, productNames[rand.Intn(len(productNames))], i+1)
		sku := fmt.Sprintf("SKU-%s-%05d", strings.ToUpper(productType[:3]), i+1)

		imagesJSON, _ := json.Marshal([]string{
			fmt.Sprintf("https://images.example.com/products/%s-1.jpg", sku),
			fmt.Sprintf("https://images.example.com/products/%s-2.jpg", sku),
		})

		product := models.Product{
			Name: name, Slug: fmt.Sprintf("%s-%s", generateSlug(name), sku),
			Description: generateLargeText(PRODUCT_DESC_SIZE, productType),
			Price:       10 + rand.Float64()*1990, CompareAtPrice: 10 + rand.Float64()*2500,
			SKU: sku, CategoryID: catIDs[rand.Intn(len(catIDs))], SellerID: sellers[rand.Intn(len(sellers))],
			Images: string(imagesJSON), Attributes: generateLargeProductAttributes(productType), Status: "active",
		}

		if err := db.Create(&product).Error; err != nil {
			continue
		}
		products = append(products, product)

		if (i+1)%100 == 0 {
			log.Printf("   ... created %d/%d products", i+1, NUM_PRODUCTS)
		}
	}

	log.Printf("   Created %d products", len(products))
	return products
}

func createAddresses(db *database.Database, users []models.User) {
	streets := []string{"Main Street", "Oak Avenue", "Maple Drive", "Cedar Lane", "Pine Road"}
	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix"}
	states := []string{"NY", "CA", "IL", "TX", "AZ"}

	count := 0
	for _, user := range users {
		if user.Role == "admin" {
			continue
		}
		cityIdx := rand.Intn(len(cities))
		address := models.Address{
			UserID: user.ID, Type: "both", FirstName: user.FirstName, LastName: user.LastName,
			AddressLine1: fmt.Sprintf("%d %s", 100+rand.Intn(9900), streets[rand.Intn(len(streets))]),
			City:         cities[cityIdx], State: states[cityIdx], PostalCode: fmt.Sprintf("%05d", 10000+rand.Intn(90000)),
			Country: "USA", IsDefault: true,
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
		{uuid.New(), "East Coast"}, {uuid.New(), "West Coast"}, {uuid.New(), "Central"},
	}

	count := 0
	for _, product := range products {
		for _, wh := range warehouses[:rand.Intn(3)+1] {
			inv := models.Inventory{
				ProductID: product.ID, WarehouseID: wh.ID, WarehouseName: wh.Name,
				Quantity: rand.Intn(500) + 10, ReservedQty: rand.Intn(50), ReorderLevel: 10, ReorderQuantity: 50,
			}
			db.Create(&inv)
			count++
		}
	}
	log.Printf("   Created %d inventory records", count)
}

func createCoupons(db *database.Database) []models.Coupon {
	now := time.Now()
	future := now.AddDate(0, 6, 0)

	coupons := []models.Coupon{
		{Code: "WELCOME10", Description: "10% off", Type: "percentage", Value: 10, MinOrderAmount: 50, IsActive: true, StartDate: &now, EndDate: &future, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		{Code: "SAVE20", Description: "$20 off", Type: "fixed", Value: 20, MinOrderAmount: 100, IsActive: true, StartDate: &now, EndDate: &future, ApplicableCategories: "[]", ApplicableProducts: "[]"},
		{Code: "FREESHIP", Description: "Free shipping", Type: "free_shipping", Value: 9.99, MinOrderAmount: 25, IsActive: true, ApplicableCategories: "[]", ApplicableProducts: "[]"},
	}

	for i := range coupons {
		db.Create(&coupons[i])
	}
	log.Printf("   Created %d coupons", len(coupons))
	return coupons
}

func createCartsWithItems(db *database.Database, users []models.User, products []models.Product) {
	count := 0
	for _, user := range users {
		if user.Role != "customer" || rand.Float32() > 0.3 {
			continue
		}
		var cart models.Cart
		if db.Where("user_id = ?", user.ID).First(&cart).Error != nil {
			continue
		}
		for j := 0; j < rand.Intn(5)+1; j++ {
			product := products[rand.Intn(len(products))]
			db.Create(&models.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: rand.Intn(3) + 1})
			count++
		}
	}
	log.Printf("   Created %d cart items", count)
}

func createOrders(db *database.Database, users []models.User, products []models.Product, coupons []models.Coupon) {
	type custAddr struct {
		UserID    uuid.UUID
		AddressID uuid.UUID
	}
	customers := []custAddr{}
	for _, u := range users {
		if u.Role == "customer" && u.Status == "active" {
			var addr models.Address
			if db.Where("user_id = ?", u.ID).First(&addr).Error == nil {
				customers = append(customers, custAddr{u.ID, addr.ID})
			}
		}
	}

	statuses := []string{"pending", "confirmed", "processing", "shipped", "delivered", "cancelled"}
	paymentMethods := []string{"card", "bank_transfer", "wallet", "cod"}

	for i := 0; i < NUM_ORDERS; i++ {
		customer := customers[rand.Intn(len(customers))]
		numItems := rand.Intn(8) + 1
		var subtotal float64

		orderItems := []struct {
			Product  models.Product
			Quantity int
		}{}
		for j := 0; j < numItems; j++ {
			product := products[rand.Intn(len(products))]
			qty := rand.Intn(3) + 1
			subtotal += product.Price * float64(qty)
			orderItems = append(orderItems, struct {
				Product  models.Product
				Quantity int
			}{product, qty})
		}

		shipping := 9.99
		if subtotal > 100 {
			shipping = 0
		}
		tax := subtotal * 0.08
		total := subtotal + shipping + tax

		status := statuses[rand.Intn(len(statuses))]

		order := models.Order{
			OrderNumber: fmt.Sprintf("ORD-%d-%05d", time.Now().Unix(), i+1),
			UserID:      customer.UserID, Status: status, Subtotal: subtotal, ShippingCost: shipping,
			Tax: tax, Total: total, ShippingAddressID: customer.AddressID, Notes: generateLargeOrderNotes(),
		}

		if db.Create(&order).Error != nil {
			continue
		}

		for _, item := range orderItems {
			db.Create(&models.OrderItem{
				OrderID: order.ID, ProductID: item.Product.ID, ProductName: item.Product.Name,
				SKU: item.Product.SKU, Quantity: item.Quantity, UnitPrice: item.Product.Price,
				TotalPrice: item.Product.Price * float64(item.Quantity),
			})
		}

		payStatus := "pending"
		if status == "delivered" || status == "shipped" {
			payStatus = "completed"
		}
		db.Create(&models.Payment{
			OrderID: order.ID, Amount: total, Currency: "USD",
			Method: paymentMethods[rand.Intn(len(paymentMethods))], Status: payStatus, GatewayResponse: "{}",
		})

		if (i+1)%500 == 0 {
			log.Printf("   ... created %d/%d orders", i+1, NUM_ORDERS)
		}
	}
	log.Printf("   Created %d orders", NUM_ORDERS)
}

func createReviews(db *database.Database, users []models.User, products []models.Product) {
	customers := []models.User{}
	for _, u := range users {
		if u.Role == "customer" && u.Status == "active" {
			customers = append(customers, u)
		}
	}

	titles := []string{"Great!", "Excellent", "Good value", "Highly recommend", "Love it!", "Five stars!"}
	reviewsCreated := 0

	for idx, product := range products {
		numReviews := REVIEWS_PER_PRODUCT/2 + rand.Intn(REVIEWS_PER_PRODUCT)
		for i := 0; i < numReviews && i < len(customers); i++ {
			customer := customers[rand.Intn(len(customers))]
			rating := 3 + rand.Intn(3) // 3-5 stars mostly

			review := models.Review{
				ProductID: product.ID, UserID: customer.ID, Rating: rating,
				Title: titles[rand.Intn(len(titles))], Content: generateLargeReviewContent(),
				Images: "[]", VerifiedPurchase: rand.Float32() < 0.7, HelpfulCount: rand.Intn(50),
			}
			if db.Create(&review).Error == nil {
				reviewsCreated++
			}
		}

		if (idx+1)%100 == 0 {
			log.Printf("   ... processed %d/%d products, created %d reviews", idx+1, len(products), reviewsCreated)
		}
	}
	log.Printf("   Created %d reviews", reviewsCreated)
}

func printSummary(db *database.Database) {
	log.Println("\n📊 Data Summary:")
	log.Println("═══════════════════════════════════════")

	tables := []string{"users", "categories", "products", "addresses", "inventories", "coupons", "carts", "cart_items", "orders", "order_items", "payments", "reviews"}
	var totalSize int64

	for _, table := range tables {
		var count int64
		db.Table(table).Count(&count)

		var size struct {
			DataLength  int64 `gorm:"column:DATA_LENGTH"`
			IndexLength int64 `gorm:"column:INDEX_LENGTH"`
		}
		db.Raw("SELECT DATA_LENGTH, INDEX_LENGTH FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", table).Scan(&size)
		tableSize := size.DataLength + size.IndexLength
		totalSize += tableSize

		log.Printf("   %-15s: %6d records (%.2f MB)", table, count, float64(tableSize)/1024/1024)
	}

	log.Println("═══════════════════════════════════════")
	log.Printf("   Total database size: %.2f MB", float64(totalSize)/1024/1024)
	log.Println("═══════════════════════════════════════")
	log.Println("\n🔑 Test Credentials:")
	log.Println("   Admin:    admin@marketplace.com / Password123!")
	log.Println("   Seller:   seller1@marketplace.com / Password123!")
	log.Println("   Customer: john.doe@email.com / Password123!")
	log.Println("")
}
