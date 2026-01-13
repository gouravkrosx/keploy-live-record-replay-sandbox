#!/usr/bin/env python3
"""
E-Commerce Marketplace API Load Testing Script
Executes 250+ API operations with configurable intervals between calls.
Designed to be re-runnable without errors.

Usage:
    python3 load_test.py                    # Default: Native/Docker mode, 250 ops, 2s delay
    python3 load_test.py --k8s              # Kubernetes mode (auto port-forward)
    python3 load_test.py --delay 1          # 1 second delay
    python3 load_test.py --delay 0.5        # 0.5 second delay
    python3 load_test.py --ops 100          # 100 operations
    python3 load_test.py --delay 1 --ops 50 # 50 ops with 1s delay
    
Native/Docker Mode (default):
    - Connects directly to localhost:1105
    - No port-forwarding needed
    
Kubernetes Mode (--k8s):
    - Automatically starts port-forward to the app service
    - Uses localhost URL via port-forward
    - Cleans up port-forward on exit
"""

import requests
import time
import json
import random
import sys
import argparse
import subprocess
import signal
import atexit
from datetime import datetime

# Configuration - can be overridden via command line
BASE_URL = "http://localhost:1105/api/v1"
DEFAULT_DELAY_SECONDS = 2
DEFAULT_TOTAL_OPERATIONS = 250
DEFAULT_NAMESPACE = "live"

# These will be set from command line args
DELAY_SECONDS = DEFAULT_DELAY_SECONDS
TOTAL_OPERATIONS = DEFAULT_TOTAL_OPERATIONS
K8S_MODE = False  # Default to native/docker mode
NAMESPACE = DEFAULT_NAMESPACE

# Port-forward process (for cleanup)
port_forward_process = None


# Test credentials (from seeded data)
ADMIN_EMAIL = "admin@marketplace.com"
ADMIN_PASSWORD = "Password123!"
CUSTOMER_EMAIL = "john.doe@email.com"
CUSTOMER_PASSWORD = "Password123!"
SELLER_EMAIL = "seller1@marketplace.com"
SELLER_PASSWORD = "Password123!"

# Colors for terminal output
class Colors:
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    BLUE = '\033[94m'
    CYAN = '\033[96m'
    RESET = '\033[0m'
    BOLD = '\033[1m'

# Global state
tokens = {
    "admin": None,
    "customer": None,
    "seller": None
}
created_resources = {
    "products": [],
    "categories": [],
    "addresses": [],
    "orders": [],
    "reviews": []
}
fetched_data = {
    "products": [],
    "categories": [],
    "users": [],
    "orders": [],
    "coupons": []
}

operation_count = 0
success_count = 0
error_count = 0

def log(message, color=Colors.RESET):
    timestamp = datetime.now().strftime("%H:%M:%S")
    print(f"{color}[{timestamp}] {message}{Colors.RESET}")

def log_operation(method, endpoint, status_code, success=True):
    global operation_count, success_count, error_count
    operation_count += 1
    
    if success:
        success_count += 1
        status_color = Colors.GREEN
        status_icon = "✓"
    else:
        error_count += 1
        status_color = Colors.RED
        status_icon = "✗"
    
    progress = f"[{operation_count}/{TOTAL_OPERATIONS}]"
    print(f"{status_color}{status_icon} {progress} {method:6} {endpoint:50} → {status_code}{Colors.RESET}")

def wait():
    time.sleep(DELAY_SECONDS)

def make_request(method, endpoint, data=None, token=None, expected_codes=[200, 201, 204]):
    """Make an HTTP request and return response"""
    url = f"{BASE_URL}{endpoint}"
    headers = {"Content-Type": "application/json"}
    
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
    try:
        if method == "GET":
            response = requests.get(url, headers=headers, timeout=30)
        elif method == "POST":
            response = requests.post(url, headers=headers, json=data, timeout=30)
        elif method == "PUT":
            response = requests.put(url, headers=headers, json=data, timeout=30)
        elif method == "DELETE":
            response = requests.delete(url, headers=headers, timeout=30)
        else:
            return None, False
        
        success = response.status_code in expected_codes
        log_operation(method, endpoint, response.status_code, success)
        
        if success and response.text:
            try:
                return response.json(), True
            except:
                return None, True
        return None, success
        
    except Exception as e:
        log_operation(method, endpoint, f"ERROR: {e}", False)
        return None, False

# ==================== Authentication Operations ====================

def login_admin():
    """Login as admin and store token"""
    data = {"email": ADMIN_EMAIL, "password": ADMIN_PASSWORD}
    response, success = make_request("POST", "/auth/login", data)
    if success and response:
        tokens["admin"] = response.get("data", {}).get("accessToken")
    wait()
    return success

def login_customer():
    """Login as customer and store token"""
    data = {"email": CUSTOMER_EMAIL, "password": CUSTOMER_PASSWORD}
    response, success = make_request("POST", "/auth/login", data)
    if success and response:
        tokens["customer"] = response.get("data", {}).get("accessToken")
    wait()
    return success

def login_seller():
    """Login as seller and store token"""
    data = {"email": SELLER_EMAIL, "password": SELLER_PASSWORD}
    response, success = make_request("POST", "/auth/login", data)
    if success and response:
        tokens["seller"] = response.get("data", {}).get("accessToken")
    wait()
    return success

def get_current_user(token_type="customer"):
    """Get current authenticated user"""
    response, success = make_request("GET", "/auth/me", token=tokens[token_type])
    wait()
    return success

def refresh_token():
    """Refresh authentication token"""
    # Just re-login for simplicity
    return login_customer()

# ==================== Product Operations ====================

def get_all_products(page=1):
    """Get all products with pagination"""
    response, success = make_request("GET", f"/products?page={page}&limit=20")
    if success and response:
        products = response.get("data", {}).get("data", [])
        for p in products[:10]:  # Store first 10
            if p.get("id") and p["id"] not in [x["id"] for x in fetched_data["products"]]:
                fetched_data["products"].append(p)
    wait()
    return success

def get_products_large_page(page=1):
    """Get products with larger page size (100 items) - causes more DB data pull"""
    response, success = make_request("GET", f"/products?page={page}&limit=100")
    if success and response:
        products = response.get("data", {}).get("data", [])
        for p in products[:20]:  # Store first 20
            if p.get("id") and p["id"] not in [x["id"] for x in fetched_data["products"]]:
                fetched_data["products"].append(p)
    wait()
    return success

def get_products_max_page(page=1):
    """Get products with max page size (200 items) - causes heavy DB data pull"""
    response, success = make_request("GET", f"/products?page={page}&limit=200")
    wait()
    return success

def get_products_with_relations():
    """Get products with category and seller relations included"""
    response, success = make_request("GET", "/products?include=category,seller&limit=50")
    wait()
    return success

def search_products(query):
    """Search products by query"""
    response, success = make_request("GET", f"/products?q={query}")
    wait()
    return success

def search_products_large(query):
    """Search products by query with large page size"""
    response, success = make_request("GET", f"/products?q={query}&limit=100")
    wait()
    return success

def filter_products_by_price(min_price, max_price):
    """Filter products by price range"""
    response, success = make_request("GET", f"/products?minPrice={min_price}&maxPrice={max_price}")
    wait()
    return success

def filter_products_by_price_large(min_price, max_price):
    """Filter products by price range with large page size - more DB pull"""
    response, success = make_request("GET", f"/products?minPrice={min_price}&maxPrice={max_price}&limit=100")
    wait()
    return success

def get_product_by_id(product_id):
    """Get single product details"""
    response, success = make_request("GET", f"/products/{product_id}")
    wait()
    return success

def get_product_with_all_data(product_id):
    """Get product with all related data - reviews, category, seller"""
    response, success = make_request("GET", f"/products/{product_id}?include=reviews,category,seller")
    wait()
    return success

def get_product_reviews(product_id):
    """Get reviews for a product"""
    response, success = make_request("GET", f"/products/{product_id}/reviews")
    wait()
    return success

def get_product_reviews_large(product_id):
    """Get reviews for a product with large page size - causes DB to pull many 2KB reviews"""
    response, success = make_request("GET", f"/products/{product_id}/reviews?limit=100")
    wait()
    return success

def get_product_reviews_max(product_id):
    """Get max reviews for a product - DB pulls ~200KB of review content"""
    response, success = make_request("GET", f"/products/{product_id}/reviews?limit=200")
    wait()
    return success

# ==================== Category Operations ====================

def get_all_categories():
    """Get category tree"""
    response, success = make_request("GET", "/categories")
    if success and response:
        categories = response.get("data", [])
        for c in categories[:10]:
            if c.get("id") and c["id"] not in [x["id"] for x in fetched_data["categories"]]:
                fetched_data["categories"].append(c)
    wait()
    return success

def get_category_by_id(category_id):
    """Get single category"""
    response, success = make_request("GET", f"/categories/{category_id}")
    wait()
    return success

# ==================== Cart Operations ====================

def get_cart():
    """Get customer cart"""
    response, success = make_request("GET", "/cart", token=tokens["customer"])
    wait()
    return success

def add_to_cart(product_id, quantity=1):
    """Add item to cart"""
    data = {"productId": product_id, "quantity": quantity}
    response, success = make_request("POST", "/cart/items", data, token=tokens["customer"])
    wait()
    return success

def validate_cart():
    """Validate cart contents"""
    response, success = make_request("POST", "/cart/validate", token=tokens["customer"])
    wait()
    return success

def clear_cart():
    """Clear the cart"""
    response, success = make_request("DELETE", "/cart", token=tokens["customer"], expected_codes=[200, 204])
    wait()
    return success

# ==================== Order Operations ====================

def get_all_orders():
    """Get all orders (admin)"""
    response, success = make_request("GET", "/orders", token=tokens["admin"])
    if success and response:
        orders = response.get("data", {}).get("data", [])
        for o in orders[:10]:
            if o.get("id") and o["id"] not in [x["id"] for x in fetched_data["orders"]]:
                fetched_data["orders"].append(o)
    wait()
    return success

def get_orders_large_page():
    """Get orders with large page size (100) - causes heavy DB pull with 1KB notes each"""
    response, success = make_request("GET", "/orders?limit=100", token=tokens["admin"])
    wait()
    return success

def get_orders_max_page():
    """Get orders with max page size (200) - causes very heavy DB pull"""
    response, success = make_request("GET", "/orders?limit=200&include=items,payment", token=tokens["admin"])
    wait()
    return success

def get_orders_with_details():
    """Get orders with all related data - items, payment, addresses"""
    response, success = make_request("GET", "/orders?include=items,payment,shippingAddress&limit=50", token=tokens["admin"])
    wait()
    return success

def get_order_by_id(order_id):
    """Get single order"""
    response, success = make_request("GET", f"/orders/{order_id}", token=tokens["admin"])
    wait()
    return success

def get_user_orders():
    """Get customer's orders - use the user ID from auth/me"""
    # First get current user ID
    response, success = make_request("GET", "/auth/me", token=tokens["customer"])
    if success and response:
        user_id = response.get("data", {}).get("id")
        if user_id:
            wait()
            response, success = make_request("GET", f"/users/{user_id}/orders", token=tokens["customer"])
            wait()
            return success
    wait()
    return True  # Don't fail if we can't get user ID

# ==================== User Operations ====================

def get_all_users():
    """Get all users (admin)"""
    response, success = make_request("GET", "/users", token=tokens["admin"])
    if success and response:
        users = response.get("data", {}).get("data", [])
        for u in users[:10]:
            if u.get("id") and u["id"] not in [x["id"] for x in fetched_data["users"]]:
                fetched_data["users"].append(u)
    wait()
    return success

def get_user_by_id(user_id):
    """Get user details (admin)"""
    response, success = make_request("GET", f"/users/{user_id}", token=tokens["admin"])
    wait()
    return success

# ==================== Coupon Operations ====================

def get_all_coupons():
    """Get all coupons (admin)"""
    response, success = make_request("GET", "/coupons", token=tokens["admin"])
    if success and response:
        coupons = response.get("data", {}).get("data", [])
        for c in coupons[:10]:
            if c.get("id") and c["id"] not in [x["id"] for x in fetched_data["coupons"]]:
                fetched_data["coupons"].append(c)
    wait()
    return success

def validate_coupon(code, cart_total=100):
    """Validate a coupon code"""
    data = {"code": code, "cartTotal": cart_total}
    response, success = make_request("POST", "/coupons/validate", data, expected_codes=[200, 400])
    wait()
    return True  # Don't fail on validation errors

# ==================== Inventory Operations ====================

def get_inventory():
    """Get inventory (admin/seller)"""
    response, success = make_request("GET", "/inventory", token=tokens["admin"])
    wait()
    return success

def get_low_stock_inventory():
    """Get low stock items"""
    response, success = make_request("GET", "/inventory?lowStock=true", token=tokens["admin"])
    wait()
    return success

# ==================== Analytics Operations ====================

def get_sales_analytics():
    """Get sales analytics"""
    response, success = make_request("GET", "/analytics/sales?fromDate=2025-01-01&toDate=2026-12-31", token=tokens["admin"])
    wait()
    return success

def get_bestsellers():
    """Get bestseller products"""
    response, success = make_request("GET", "/analytics/products/bestsellers?limit=10", token=tokens["admin"])
    wait()
    return success

def get_customer_analytics():
    """Get customer analytics"""
    response, success = make_request("GET", "/analytics/customers?fromDate=2025-01-01&toDate=2026-12-31", token=tokens["admin"])
    wait()
    return success

def get_inventory_alerts():
    """Get inventory alerts"""
    response, success = make_request("GET", "/analytics/inventory/alerts", token=tokens["admin"])
    wait()
    return success

# ==================== Health Check ====================

def health_check():
    """Check API health"""
    try:
        response = requests.get(f"{BASE_URL.replace('/api/v1', '')}/health", timeout=10)
        success = response.status_code == 200
        log_operation("GET", "/health", response.status_code, success)
        wait()
        return success
    except Exception as e:
        log_operation("GET", "/health", f"ERROR: {e}", False)
        wait()
        return False

# ==================== Main Script ====================

def run_operations():
    """Execute all operations to reach target count"""
    global operation_count
    
    log(f"\n{'='*60}", Colors.BOLD)
    log("E-Commerce Marketplace API Load Test", Colors.BOLD)
    log(f"Target: {TOTAL_OPERATIONS} operations with {DELAY_SECONDS}s delay", Colors.CYAN)
    log(f"{'='*60}\n", Colors.BOLD)
    
    # Phase 1: Health check
    log("Phase 1: Health Check", Colors.YELLOW)
    if not health_check():
        log("API is not healthy. Exiting.", Colors.RED)
        return False
    
    # Phase 2: Authentication
    log("\nPhase 2: Authentication", Colors.YELLOW)
    login_admin()
    login_customer()
    login_seller()
    
    if not tokens["admin"] or not tokens["customer"]:
        log("Failed to authenticate. Exiting.", Colors.RED)
        return False
    
    # Phase 3: Initial data fetch
    log("\nPhase 3: Fetching Initial Data", Colors.YELLOW)
    get_all_categories()
    get_all_products()
    get_all_products(page=2)
    get_all_users()
    get_all_orders()
    get_all_coupons()
    
    # Phase 4: Read operations loop
    log("\nPhase 4: Read Operations (Heavy Data Pull)", Colors.YELLOW)
    
    operations = [
        # Product operations - normal
        lambda: get_all_products(random.randint(1, 3)),
        lambda: search_products(random.choice(["laptop", "phone", "shirt", "shoes", "camera", "premium", "pro", "ultra"])),
        lambda: filter_products_by_price(50, 500),
        lambda: filter_products_by_price(100, 1000),
        lambda: filter_products_by_price(500, 5000),
        lambda: get_product_by_id(random.choice(fetched_data["products"])["id"]) if fetched_data["products"] else get_all_products(),
        lambda: get_product_reviews(random.choice(fetched_data["products"])["id"]) if fetched_data["products"] else get_all_products(),
        
        # Product operations - HEAVY DATA PULL (large pages, relations)
        lambda: get_products_large_page(random.randint(1, 10)),
        lambda: get_products_max_page(random.randint(1, 5)),
        lambda: get_products_with_relations(),
        lambda: search_products_large(random.choice(["pro", "premium", "ultra", "deluxe", "edition"])),
        lambda: filter_products_by_price_large(10, 2000),
        lambda: get_product_with_all_data(random.choice(fetched_data["products"])["id"]) if fetched_data["products"] else get_all_products(),
        lambda: get_product_reviews_large(random.choice(fetched_data["products"])["id"]) if fetched_data["products"] else get_all_products(),
        lambda: get_product_reviews_max(random.choice(fetched_data["products"])["id"]) if fetched_data["products"] else get_all_products(),
        
        # Category operations
        lambda: get_all_categories(),
        lambda: get_category_by_id(random.choice(fetched_data["categories"])["id"]) if fetched_data["categories"] else get_all_categories(),
        
        # Cart operations
        lambda: get_cart(),
        lambda: add_to_cart(random.choice(fetched_data["products"])["id"]) if fetched_data["products"] else get_cart(),
        lambda: validate_cart(),
        lambda: clear_cart(),
        
        # User operations
        lambda: get_current_user("customer"),
        lambda: get_current_user("admin"),
        lambda: get_all_users(),
        lambda: get_user_by_id(random.choice(fetched_data["users"])["id"]) if fetched_data["users"] else get_all_users(),
        
        # Order operations - normal
        lambda: get_all_orders(),
        lambda: get_order_by_id(random.choice(fetched_data["orders"])["id"]) if fetched_data["orders"] else get_all_orders(),
        lambda: get_user_orders(),
        
        # Order operations - HEAVY DATA PULL (large pages, relations)
        lambda: get_orders_large_page(),
        lambda: get_orders_max_page(),
        lambda: get_orders_with_details(),
        
        # Coupon operations
        lambda: get_all_coupons(),
        lambda: validate_coupon("WELCOME10"),
        lambda: validate_coupon("SAVE20"),
        lambda: validate_coupon("INVALID123"),  # Test invalid coupon
        
        # Inventory operations
        lambda: get_inventory(),
        lambda: get_low_stock_inventory(),
        
        # Analytics operations
        lambda: get_sales_analytics(),
        lambda: get_bestsellers(),
        lambda: get_customer_analytics(),
        lambda: get_inventory_alerts(),
        
        # Auth operations
        lambda: get_current_user("seller"),
        lambda: refresh_token(),
    ]
    
    # Keep running operations until we reach the target
    while operation_count < TOTAL_OPERATIONS:
        try:
            # Pick a random operation
            operation = random.choice(operations)
            operation()
        except Exception as e:
            log(f"Operation error: {e}", Colors.RED)
            wait()
    
    return True

def print_summary():
    """Print execution summary"""
    log(f"\n{'='*60}", Colors.BOLD)
    log("Execution Summary", Colors.BOLD)
    log(f"{'='*60}", Colors.BOLD)
    log(f"Total Operations: {operation_count}", Colors.CYAN)
    log(f"Successful:       {success_count} ({100*success_count//max(operation_count,1)}%)", Colors.GREEN)
    log(f"Failed:           {error_count} ({100*error_count//max(operation_count,1)}%)", Colors.RED if error_count > 0 else Colors.GREEN)
    log(f"{'='*60}\n", Colors.BOLD)

def parse_args():
    """Parse command line arguments"""
    parser = argparse.ArgumentParser(
        description="E-Commerce Marketplace API Load Testing Script",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    python3 load_test.py                     # Default: Native/Docker mode, 250 ops, 2s delay
    python3 load_test.py --k8s               # Kubernetes mode (auto port-forward)
    python3 load_test.py --delay 1           # 1 second delay
    python3 load_test.py --delay 0.5         # 0.5 second delay  
    python3 load_test.py --ops 100           # 100 operations
    python3 load_test.py --delay 1 --ops 50  # 50 ops with 1s delay
    python3 load_test.py -d 0.5 -o 100       # Short form
    python3 load_test.py --k8s -n custom     # K8s mode with custom namespace

Native/Docker Mode (default):
    Connect directly to localhost:1105 without port-forwarding.
    
Kubernetes Mode (--k8s):
    The script will automatically set up port-forwarding to the
    marketplace-api service in the specified namespace.
        """
    )
    parser.add_argument(
        "-d", "--delay",
        type=float,
        default=DEFAULT_DELAY_SECONDS,
        help=f"Delay in seconds between API calls (default: {DEFAULT_DELAY_SECONDS})"
    )
    parser.add_argument(
        "-o", "--ops",
        type=int,
        default=DEFAULT_TOTAL_OPERATIONS,
        help=f"Total number of operations to perform (default: {DEFAULT_TOTAL_OPERATIONS})"
    )
    parser.add_argument(
        "-u", "--url",
        type=str,
        default=BASE_URL,
        help=f"Base URL for the API (default: {BASE_URL})"
    )
    parser.add_argument(
        "--k8s",
        action="store_true",
        default=False,
        dest="k8s",
        help="Enable Kubernetes mode with auto port-forwarding (default: disabled)"
    )
    parser.add_argument(
        "-n", "--namespace",
        type=str,
        default=DEFAULT_NAMESPACE,
        help=f"Kubernetes namespace, used with --k8s (default: {DEFAULT_NAMESPACE})"
    )
    return parser.parse_args()

def cleanup_port_forward():
    """Clean up port-forward process"""
    global port_forward_process
    if port_forward_process:
        log("Stopping port-forward...", Colors.YELLOW)
        port_forward_process.terminate()
        try:
            port_forward_process.wait(timeout=5)
        except:
            port_forward_process.kill()
        port_forward_process = None

def start_port_forward(namespace):
    """Start kubectl port-forward in background"""
    global port_forward_process
    
    log(f"Starting port-forward to marketplace-api in namespace '{namespace}'...", Colors.CYAN)
    
    try:
        pod_cmd = ["kubectl", "get", "pods", "-n", namespace, "-l", "app=marketplace-api", "-o", "jsonpath={.items[0].metadata.name}"]
        pod_name = subprocess.check_output(pod_cmd).decode().strip()
        
        if not pod_name:
            log("No pods found!", Colors.RED)
            return False

        log(f"Targeting specific pod: {pod_name}", Colors.YELLOW)

        # 2. PORT FORWARD TO THAT SPECIFIC POD (Instead of the Service)
        port_forward_process = subprocess.Popen(
            ["kubectl", "port-forward", f"pod/{pod_name}", "1105:1105", "-n", namespace],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        
        # Register cleanup handler
        atexit.register(cleanup_port_forward)
        
        # Wait for port-forward to be ready
        log("Waiting for port-forward to be ready...", Colors.CYAN)
        time.sleep(3)
        
        # Check if process is still running
        if port_forward_process.poll() is not None:
            stderr = port_forward_process.stderr.read().decode()
            log(f"Port-forward failed: {stderr}", Colors.RED)
            return False
        
        # Test connection
        try:
            response = requests.get("http://localhost:1105/health", timeout=5)
            if response.status_code == 200:
                log("✓ Port-forward ready!", Colors.GREEN)
                return True
        except:
            pass
        
        # Give it a bit more time
        time.sleep(2)
        try:
            response = requests.get("http://localhost:1105/health", timeout=5)
            if response.status_code == 200:
                log("✓ Port-forward ready!", Colors.GREEN)
                return True
        except Exception as e:
            log(f"Could not connect: {e}", Colors.RED)
            return False
            
    except FileNotFoundError:
        log("kubectl not found. Please install kubectl.", Colors.RED)
        return False
    except Exception as e:
        log(f"Failed to start port-forward: {e}", Colors.RED)
        return False

def main():
    global DELAY_SECONDS, TOTAL_OPERATIONS, BASE_URL, K8S_MODE, NAMESPACE
    
    # Parse command line arguments
    args = parse_args()
    DELAY_SECONDS = args.delay
    TOTAL_OPERATIONS = args.ops
    BASE_URL = args.url
    K8S_MODE = args.k8s
    NAMESPACE = args.namespace
    
    # Handle K8s mode
    if K8S_MODE:
        log(f"\n{'='*60}", Colors.BOLD)
        log("Kubernetes Mode Enabled", Colors.CYAN)
        log(f"{'='*60}", Colors.BOLD)
        
        if not start_port_forward(NAMESPACE):
            log("Failed to set up port-forward. Exiting.", Colors.RED)
            log("Hint: Use --no-k8s flag for native/docker mode.", Colors.YELLOW)
            return 1
    else:
        log(f"\n{'='*60}", Colors.BOLD)
        log("Native/Docker Mode (no port-forward)", Colors.CYAN)
        log(f"{'='*60}", Colors.BOLD)
    
    start_time = time.time()
    
    try:
        success = run_operations()
        print_summary()
        
        elapsed = time.time() - start_time
        log(f"Total time: {elapsed:.1f} seconds ({elapsed/60:.1f} minutes)", Colors.CYAN)
        
        if error_count == 0:
            log("All operations completed successfully! ✓", Colors.GREEN)
            return 0
        else:
            log(f"Completed with {error_count} errors", Colors.YELLOW)
            return 1
            
    except KeyboardInterrupt:
        log("\n\nInterrupted by user", Colors.YELLOW)
        print_summary()
        return 130
    except Exception as e:
        log(f"\nFatal error: {e}", Colors.RED)
        return 1
    finally:
        # Clean up port-forward
        if K8S_MODE:
            cleanup_port_forward()

if __name__ == "__main__":
    sys.exit(main())

