#!/usr/bin/env python3
# for large size mocks
"""
Heavy Data Load Test Script - Tests heavy database operation endpoints
NOT part of regular application flow - used for testing large mock scenarios

Usage:
    python3 heavy_load_test.py                    # Default: 10 APIs, native/docker
    python3 heavy_load_test.py --api 20           # Hit all 20 APIs
    python3 heavy_load_test.py --api 5            # Hit 5 APIs
    python3 heavy_load_test.py --k8s              # Use k8s port-forward URL
    python3 heavy_load_test.py --api 20 --k8s     # All 20 APIs on k8s
"""

import requests
import time
import argparse
import subprocess
import signal
import atexit
import sys
import json
from datetime import datetime

# for large size mocks - Configuration
NATIVE_BASE_URL = "http://localhost:1105/api/v1"
K8S_BASE_URL = "http://localhost:1105/api/v1"  # Via auto port-forward
DEFAULT_NAMESPACE = "live"

ADMIN_EMAIL = "admin@marketplace.com"
ADMIN_PASSWORD = "Password123!"

# Port-forward process (for cleanup)
port_forward_process = None

# Colors for terminal output
class Colors:
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    BLUE = '\033[94m'
    CYAN = '\033[96m'
    RESET = '\033[0m'
    BOLD = '\033[1m'

def log(message, color=Colors.RESET):
    timestamp = datetime.now().strftime("%H:%M:%S")
    print(f"{color}[{timestamp}] {message}{Colors.RESET}")

def login(base_url):
    """Login as admin and return access token"""
    log("🔐 Authenticating as admin...", Colors.YELLOW)
    try:
        response = requests.post(
            f"{base_url}/auth/login",
            json={"email": ADMIN_EMAIL, "password": ADMIN_PASSWORD},
            timeout=30
        )
        if response.status_code == 200:
            token = response.json().get("data", {}).get("accessToken")
            log("✓ Authentication successful", Colors.GREEN)
            return token
        else:
            log(f"✗ Authentication failed: {response.status_code}", Colors.RED)
            return None
    except Exception as e:
        log(f"✗ Authentication error: {e}", Colors.RED)
        return None

def call_heavy_endpoint(base_url, endpoint, token):
    """Call a heavy endpoint and return results"""
    url = f"{base_url}/heavy/{endpoint}"
    headers = {"Authorization": f"Bearer {token}"}
    
    try:
        start = time.time()
        response = requests.get(url, headers=headers, timeout=120)
        elapsed = time.time() - start
        
        if response.status_code == 200:
            data = response.json()
            return {
                "success": True,
                "endpoint": endpoint,
                "dataPulledMB": data.get("dataPulledMB", 0),
                "recordCount": data.get("recordCount", 0),
                "tablesAccessed": data.get("tablesAccessed", []),
                "queryTimeMs": data.get("queryTimeMs", 0),
                "clientTimeMs": int(elapsed * 1000),
                "message": data.get("message", "")
            }
        else:
            return {
                "success": False,
                "endpoint": endpoint,
                "error": f"HTTP {response.status_code}"
            }
    except Exception as e:
        return {
            "success": False,
            "endpoint": endpoint,
            "error": str(e)
        }

def run_heavy_tests(base_url, num_apis, is_k8s):
    """Run heavy endpoint tests"""
    env_name = "Kubernetes" if is_k8s else "Native/Docker"
    
    log(f"\n{'='*60}", Colors.BOLD)
    log("Heavy Data Load Test - Large Database Operations", Colors.BOLD)
    log(f"Environment: {env_name}", Colors.CYAN)
    log(f"Target: {base_url}", Colors.CYAN)
    log(f"APIs to test: {num_apis}", Colors.CYAN)
    log(f"{'='*60}\n", Colors.BOLD)
    
    # Authenticate
    token = login(base_url)
    if not token:
        log("Failed to authenticate. Exiting.", Colors.RED)
        return False
    
    # for large size mocks - All 20 heavy endpoints available
    all_endpoints = [
        # Original 10 endpoints
        "products",
        "orders", 
        "reviews",
        "inventory",
        "aggregate",
        "users",
        "categories",
        "payments",
        "carts",
        "full-dump",
        # New 10 endpoints
        "product-search",
        "order-history",
        "user-activity",
        "analytics-dashboard",
        "sales-trends",
        "inventory-report",
        "review-sentiment",
        "category-tree",
        "shipping-data",
        "financial-summary"
    ]
    
    # Select endpoints based on --api flag
    endpoints = all_endpoints[:num_apis]
    
    results = []
    total_data_mb = 0
    total_records = 0
    
    log(f"\n📊 Executing {len(endpoints)} Heavy Endpoints...\n", Colors.YELLOW)
    
    for i, endpoint in enumerate(endpoints, 1):
        log(f"[{i}/{len(endpoints)}] Testing /heavy/{endpoint}...", Colors.BLUE)
        
        result = call_heavy_endpoint(base_url, endpoint, token)
        results.append(result)
        
        if result["success"]:
            total_data_mb += result["dataPulledMB"]
            total_records += result["recordCount"]
            log(f"  ✓ Data: {result['dataPulledMB']:.2f}MB | Records: {result['recordCount']} | Query: {result['queryTimeMs']}ms | Client: {result['clientTimeMs']}ms", Colors.GREEN)
        else:
            log(f"  ✗ Error: {result.get('error', 'Unknown error')}", Colors.RED)
        
        time.sleep(0.5)  # Brief pause between requests
    
    # Print summary
    log(f"\n{'='*60}", Colors.BOLD)
    log("Summary", Colors.BOLD)
    log(f"{'='*60}", Colors.BOLD)
    
    successful = sum(1 for r in results if r["success"])
    failed = len(results) - successful
    
    log(f"Environment:         {env_name}", Colors.CYAN)
    log(f"Endpoints tested:    {len(endpoints)}/{len(all_endpoints)}", Colors.CYAN)
    log(f"Successful:          {successful}", Colors.GREEN)
    log(f"Failed:              {failed}", Colors.RED if failed > 0 else Colors.GREEN)
    log(f"Total data pulled:   {total_data_mb:.2f} MB", Colors.CYAN)
    log(f"Total records:       {total_records}", Colors.CYAN)
    
    log(f"\n{'='*60}\n", Colors.BOLD)
    
    # Detailed results table
    log("Detailed Results:", Colors.YELLOW)
    print(f"{'Endpoint':<20} {'Data (MB)':<12} {'Records':<10} {'Query (ms)':<12} {'Status':<10}")
    print("-" * 70)
    
    for r in results:
        if r["success"]:
            status = "✓ OK"
            print(f"{r['endpoint']:<20} {r['dataPulledMB']:<12.2f} {r['recordCount']:<10} {r['queryTimeMs']:<12} {status}")
        else:
            print(f"{r['endpoint']:<20} {'N/A':<12} {'N/A':<10} {'N/A':<12} ✗ FAILED")
    
    print()
    return failed == 0

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
        # Get all pods with app=marketplace-api label in JSON format
        pod_cmd = ["kubectl", "get", "pods", "-n", namespace, "-l", "app=marketplace-api", "-o", "json"]
        pod_output = subprocess.check_output(pod_cmd).decode().strip()
        pod_data = json.loads(pod_output)
        
        target_pod_name = None
        fallback_pod_name = None
        
        items = pod_data.get("items", [])
        if not items:
            log("No pods found!", Colors.RED)
            return False

        # Iterate through pods to find the best candidate
        target_pod_name = None
        fallback_pod_name = None
        
        for pod in items:
            metadata = pod.get("metadata", {})
            status = pod.get("status", {})
            spec = pod.get("spec", {})
            
            name = metadata.get("name")
            phase = status.get("phase")
            
            if phase == "Running":
                # If we haven't found a fallback yet, use this one
                if not fallback_pod_name:
                    fallback_pod_name = name
                
                # Check Init Containers for keploy-agent with mode record
                init_containers = spec.get("initContainers", [])
                for ic in init_containers:
                    if ic.get("name") == "keploy-agent":
                        args = ic.get("args", [])
                        is_record_mode = False
                        
                        args_str = " ".join(args)
                        if "--mode record" in args_str or "--mode=record" in args_str:
                             is_record_mode = True
                        
                        try:
                            if "--mode" in args:
                                idx = args.index("--mode")
                                if idx + 1 < len(args) and args[idx+1] == "record":
                                    is_record_mode = True
                        except:
                            pass

                        if is_record_mode:
                            target_pod_name = name
                            log(f"Found Keploy Record Pod: {name}", Colors.BLUE)
                            break
                
                if target_pod_name:
                    break # Found it
                    
        # Select the pod to use
        final_pod_name = target_pod_name if target_pod_name else fallback_pod_name
        
        if not final_pod_name:
            log("No running pods found!", Colors.RED)
            return False

        if target_pod_name:
             log(f"Targeting Keploy Record Pod: {final_pod_name}", Colors.YELLOW)
        else:
             log(f"Targeting available pod: {final_pod_name}", Colors.YELLOW)
             log("(Warning: Could not find pod with 'keploy-agent' in record mode)", Colors.YELLOW)

        # Start port-forward in background
        port_forward_process = subprocess.Popen(
            ["kubectl", "port-forward", f"pod/{final_pod_name}", "1105:1105", "-n", namespace],
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
    except json.JSONDecodeError:
        log("Failed to parse kubectl output.", Colors.RED)
        return False
    except Exception as e:
        log(f"Failed to start port-forward: {e}", Colors.RED)
        return False

def parse_args():
    parser = argparse.ArgumentParser(
        description="Heavy Data Load Test - Tests large database operation endpoints",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    # Native/Docker: Test 10 APIs (default)
    python3 heavy_load_test.py
    
    # Native/Docker: Test all 20 APIs
    python3 heavy_load_test.py --api 20
    
    # Native/Docker: Test 5 APIs
    python3 heavy_load_test.py --api 5
    
    # Kubernetes: Test 10 APIs (auto port-forward)
    python3 heavy_load_test.py --k8s
    
    # Kubernetes: Test all 20 APIs
    python3 heavy_load_test.py --api 20 --k8s
    
    # Kubernetes with custom namespace
    python3 heavy_load_test.py --k8s -n custom-namespace

Note: 
  - For native/docker, ensure server is running on localhost:1105
  - For k8s, port-forward is automatically set up and cleaned up
  - Make sure to run the seeder first to populate test data
        """
    )
    parser.add_argument(
        "--api",
        type=int,
        default=10,
        choices=range(1, 21),
        metavar="N",
        help="Number of APIs to hit (1-20, default: 10)"
    )
    parser.add_argument(
        "--k8s",
        action="store_true",
        help="Use Kubernetes environment with auto port-forwarding"
    )
    parser.add_argument(
        "-n", "--namespace",
        type=str,
        default=DEFAULT_NAMESPACE,
        help=f"Kubernetes namespace, used with --k8s (default: {DEFAULT_NAMESPACE})"
    )
    return parser.parse_args()

def main():
    args = parse_args()
    
    # Handle K8s mode with auto port-forwarding
    if args.k8s:
        log(f"\n{'='*60}", Colors.BOLD)
        log("Kubernetes Mode - Auto Port-Forward", Colors.CYAN)
        log(f"{'='*60}", Colors.BOLD)
        
        if not start_port_forward(args.namespace):
            log("Failed to set up port-forward. Exiting.", Colors.RED)
            return 1
    else:
        log(f"\n{'='*60}", Colors.BOLD)
        log("Native/Docker Mode (no port-forward)", Colors.CYAN)
        log(f"{'='*60}", Colors.BOLD)
    
    # Select base URL based on environment
    base_url = K8S_BASE_URL if args.k8s else NATIVE_BASE_URL
    
    start_time = time.time()
    
    try:
        success = run_heavy_tests(base_url, args.api, args.k8s)
        elapsed = time.time() - start_time
        
        log(f"Total execution time: {elapsed:.1f} seconds", Colors.CYAN)
        
        if success:
            log("All heavy endpoint tests completed successfully! ✓", Colors.GREEN)
            return 0
        else:
            log("Some tests failed", Colors.YELLOW)
            return 1
    except KeyboardInterrupt:
        log("\n\nInterrupted by user", Colors.YELLOW)
        return 130
    except Exception as e:
        log(f"\nFatal error: {e}", Colors.RED)
        return 1
    finally:
        # Clean up port-forward if in K8s mode
        if args.k8s:
            cleanup_port_forward()

if __name__ == "__main__":
    sys.exit(main())

