# API Flow Diagrams

This document shows the flow of API calls for common user journeys in the E-Commerce Marketplace API.

## Table of Contents
1. [Authentication Flow](#1-authentication-flow)
2. [Product Browsing Flow](#2-product-browsing-flow)
3. [Shopping Cart Flow](#3-shopping-cart-flow)
4. [Checkout Flow](#4-checkout-flow)
5. [Order Management Flow](#5-order-management-flow)
6. [Complete E-Commerce Journey](#6-complete-e-commerce-journey)

---

## 1. Authentication Flow

### User Registration & Login

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           AUTHENTICATION FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐                                              ┌──────────────┐
    │  Client  │                                              │   Server     │
    └────┬─────┘                                              └──────┬───────┘
         │                                                           │
         │  ══════════════ REGISTRATION ══════════════              │
         │                                                           │
         │  POST /api/v1/auth/register                              │
         │  {email, password, firstName, lastName}                   │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    201 Created                            │
         │  {user, accessToken, refreshToken, expiresIn}            │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ════════════════ LOGIN ════════════════                 │
         │                                                           │
         │  POST /api/v1/auth/login                                 │
         │  {email, password}                                        │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {user, accessToken, refreshToken, expiresIn}            │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ════════════ TOKEN REFRESH ════════════                 │
         │                                                           │
         │  POST /api/v1/auth/refresh                               │
         │  {refreshToken}                                           │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {accessToken, refreshToken, expiresIn}                  │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ════════════════ LOGOUT ════════════════                │
         │                                                           │
         │  POST /api/v1/auth/logout                                │
         │  Authorization: Bearer <accessToken>                      │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {message: "Successfully logged out"}                    │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
```

### Password Reset Flow

```
    ┌──────────┐                                              ┌──────────────┐
    │  Client  │                                              │   Server     │
    └────┬─────┘                                              └──────┬───────┘
         │                                                           │
         │  POST /api/v1/auth/password/reset                        │
         │  {email}                                                  │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {message, resetToken}                                    │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  POST /api/v1/auth/password/reset/confirm                │
         │  {token, newPassword}                                     │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {message: "Password reset successful"}                  │
         │◄──────────────────────────────────────────────────────────┤
```

---

## 2. Product Browsing Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         PRODUCT BROWSING FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐                                              ┌──────────────┐
    │  Client  │                                              │   Server     │
    └────┬─────┘                                              └──────┬───────┘
         │                                                           │
         │  ════════════ GET CATEGORIES ════════════                │
         │                                                           │
         │  GET /api/v1/categories?includeProductCount=true         │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  [{id, name, children: [...], productCount}]             │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ════════════ SEARCH PRODUCTS ════════════               │
         │                                                           │
         │  GET /api/v1/products?q=laptop&categoryId=xxx            │
         │      &minPrice=100&maxPrice=1000&inStock=true            │
         │      &sortBy=price_asc&page=1&limit=20                   │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {data: [...products], pagination: {...}}                │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ═══════════ GET PRODUCT DETAILS ═══════════             │
         │                                                           │
         │  GET /api/v1/products/{productId}                        │
         │      ?include=category,seller,reviews,inventory          │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {product, reviewStats, totalStock, relatedProducts}     │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ════════════ GET REVIEWS ════════════                   │
         │                                                           │
         │  GET /api/v1/products/{productId}/reviews                │
         │      ?sortBy=most_helpful&verified=true                  │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {reviews: [...], stats: {avgRating, distribution}}      │
         │◄──────────────────────────────────────────────────────────┤
```

---

## 3. Shopping Cart Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SHOPPING CART FLOW                                 │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐                                              ┌──────────────┐
    │  Client  │               [Authenticated]                │   Server     │
    └────┬─────┘                                              └──────┬───────┘
         │                                                           │
         │  ═══════════════ ADD TO CART ═══════════════             │
         │                                                           │
         │  POST /api/v1/cart/items                                 │
         │  Authorization: Bearer <token>                            │
         │  {productId, quantity: 2}                                │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {cart with all items and totals}                        │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ════════════════ VIEW CART ════════════════             │
         │                                                           │
         │  GET /api/v1/cart                                        │
         │  Authorization: Bearer <token>                            │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {items, subtotal, discount, shipping, tax, total}       │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ═══════════ UPDATE ITEM QUANTITY ═══════════            │
         │                                                           │
         │  PUT /api/v1/cart/items/{itemId}                         │
         │  {quantity: 3}                                            │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {updated cart}                                          │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ════════════ APPLY COUPON ════════════                  │
         │                                                           │
         │  POST /api/v1/cart/apply-coupon                          │
         │  {code: "SAVE10"}                                        │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {cart with discount applied}                            │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ═══════════ VALIDATE CART ═══════════                   │
         │                                                           │
         │  POST /api/v1/cart/validate                              │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {valid: true/false, issues: [...]}                      │
         │◄──────────────────────────────────────────────────────────┤
```

---

## 4. Checkout Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             CHECKOUT FLOW                                    │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐                                              ┌──────────────┐
    │  Client  │               [Authenticated]                │   Server     │
    └────┬─────┘                                              └──────┬───────┘
         │                                                           │
         │  ═══════════ GET/ADD ADDRESS ═══════════                 │
         │                                                           │
         │  GET /api/v1/users/{userId}/addresses                    │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  [addresses]                                              │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  POST /api/v1/users/{userId}/addresses  (if needed)      │
         │  {firstName, lastName, addressLine1, city, ...}          │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    201 Created                            │
         │  {new address}                                            │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ═══════════ VALIDATE CART ═══════════                   │
         │                                                           │
         │  POST /api/v1/cart/validate                              │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │                    200 OK                                 │
         │  {valid: true}                                            │
         │◄──────────────────────────────────────────────────────────┤
         │                                                           │
         │  ═════════════ CREATE ORDER ═════════════                │
         │                                                           │
         │  POST /api/v1/orders                                     │
         │  {                                                        │
         │    shippingAddressId,                                    │
         │    billingAddressId,                                     │
         │    paymentMethod: "card",                                │
         │    couponCode: "SAVE10"                                  │
         │  }                                                        │
         ├──────────────────────────────────────────────────────────►│
         │                                                           │
         │     ┌─────────────────────────────────────────┐          │
         │     │ Server Processing:                       │          │
         │     │ 1. Validate cart items                   │          │
         │     │ 2. Calculate totals                      │          │
         │     │ 3. Apply coupon                          │          │
         │     │ 4. Create order & order items           │          │
         │     │ 5. Reserve inventory                     │          │
         │     │ 6. Create payment record                 │          │
         │     │ 7. Clear cart                            │          │
         │     └─────────────────────────────────────────┘          │
         │                                                           │
         │                    201 Created                            │
         │  {order with items, payment, addresses}                  │
         │◄──────────────────────────────────────────────────────────┤
```

---

## 5. Order Management Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ORDER MANAGEMENT FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐        ┌──────────┐         ┌────────────┐    ┌─────────────┐
    │ Customer │        │  Admin   │         │  Server    │    │  Database   │
    └────┬─────┘        └────┬─────┘         └─────┬──────┘    └──────┬──────┘
         │                   │                     │                   │
         │  ══════════ VIEW ORDER HISTORY ══════════                  │
         │                   │                     │                   │
         │  GET /api/v1/users/{userId}/orders     │                   │
         ├────────────────────────────────────────►│                   │
         │                   │                     ├──────────────────►│
         │                   │                     │  Query orders     │
         │                   │                     │◄──────────────────┤
         │  {orders, stats, pagination}           │                   │
         │◄────────────────────────────────────────┤                   │
         │                   │                     │                   │
         │                   │  ══════ UPDATE ORDER STATUS ══════     │
         │                   │                     │                   │
         │                   │  PUT /api/v1/orders/{orderId}/status   │
         │                   │  {status: "shipped", trackingNumber}   │
         │                   ├────────────────────►│                   │
         │                   │                     ├──────────────────►│
         │                   │                     │  Update order     │
         │                   │                     │◄──────────────────┤
         │                   │  {updated order}    │                   │
         │                   │◄────────────────────┤                   │
         │                   │                     │                   │
         │  ═══════════ CANCEL ORDER ═══════════  │                   │
         │                   │                     │                   │
         │  POST /api/v1/orders/{orderId}/cancel  │                   │
         ├────────────────────────────────────────►│                   │
         │                   │                     │                   │
         │     ┌─────────────────────────────────────────┐            │
         │     │ Server Processing:                       │            │
         │     │ 1. Validate cancellation allowed         │            │
         │     │ 2. Update order status                   │            │
         │     │ 3. Release reserved inventory            │            │
         │     └─────────────────────────────────────────┘            │
         │                   │                     │                   │
         │  {message: "Order cancelled"}          │                   │
         │◄────────────────────────────────────────┤                   │
         │                   │                     │                   │


  ORDER STATUS LIFECYCLE:
  ═══════════════════════

    ┌─────────┐     ┌───────────┐     ┌────────────┐     ┌─────────┐     ┌───────────┐
    │ PENDING │────►│ CONFIRMED │────►│ PROCESSING │────►│ SHIPPED │────►│ DELIVERED │
    └────┬────┘     └───────────┘     └────────────┘     └─────────┘     └───────────┘
         │                                                                      │
         │                                                                      │
         ▼                                                                      ▼
    ┌───────────┐                                                        ┌──────────┐
    │ CANCELLED │                                                        │ REFUNDED │
    └───────────┘                                                        └──────────┘
```

---

## 6. Complete E-Commerce Journey

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      COMPLETE E-COMMERCE JOURNEY                             │
└─────────────────────────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────────────────────┐
  │                                                                         │
  │  1. REGISTRATION                                                        │
  │     POST /api/v1/auth/register                                         │
  │     └── Returns: user + tokens                                         │
  │                    │                                                    │
  │                    ▼                                                    │
  │  2. BROWSE PRODUCTS                                                     │
  │     GET /api/v1/categories                                             │
  │     GET /api/v1/products?q=...&categoryId=...                         │
  │     GET /api/v1/products/{id}                                          │
  │     GET /api/v1/products/{id}/reviews                                  │
  │                    │                                                    │
  │                    ▼                                                    │
  │  3. ADD TO CART                                                         │
  │     POST /api/v1/cart/items {productId, quantity}                      │
  │     PUT /api/v1/cart/items/{id} {quantity}                             │
  │     POST /api/v1/cart/apply-coupon {code}                              │
  │                    │                                                    │
  │                    ▼                                                    │
  │  4. CHECKOUT                                                            │
  │     GET /api/v1/users/{id}/addresses                                   │
  │     POST /api/v1/users/{id}/addresses (if needed)                      │
  │     POST /api/v1/cart/validate                                         │
  │     POST /api/v1/orders                                                │
  │                    │                                                    │
  │                    ▼                                                    │
  │  5. ORDER TRACKING                                                      │
  │     GET /api/v1/orders/{orderId}                                       │
  │     GET /api/v1/users/{id}/orders                                      │
  │                    │                                                    │
  │                    ▼                                                    │
  │  6. POST-PURCHASE                                                       │
  │     POST /api/v1/products/{id}/reviews (after delivery)                │
  │     POST /api/v1/orders/{id}/cancel (if needed)                        │
  │     POST /api/v1/orders/{id}/refund (if needed)                        │
  │                                                                         │
  └─────────────────────────────────────────────────────────────────────────┘


  PARALLEL ADMIN OPERATIONS:
  ══════════════════════════

  ┌──────────────────────────────────┐
  │ INVENTORY MANAGEMENT             │
  │ GET /api/v1/inventory            │
  │ PUT /api/v1/inventory/{id}       │
  │ POST /api/v1/inventory/transfer  │
  │ POST /api/v1/inventory/adjustments│
  └──────────────────────────────────┘

  ┌──────────────────────────────────┐
  │ COUPON MANAGEMENT                │
  │ GET /api/v1/coupons              │
  │ POST /api/v1/coupons             │
  │ PUT /api/v1/coupons/{id}         │
  │ POST /api/v1/coupons/validate    │
  └──────────────────────────────────┘

  ┌──────────────────────────────────┐
  │ ANALYTICS                        │
  │ GET /api/v1/analytics/sales      │
  │ GET /api/v1/analytics/bestsellers│
  │ GET /api/v1/analytics/customers  │
  │ GET /api/v1/analytics/inventory/alerts│
  └──────────────────────────────────┘
```

---

## API Endpoint Quick Reference

| Category | Method | Endpoint | Auth Required |
|----------|--------|----------|---------------|
| **Auth** | POST | /api/v1/auth/register | ❌ |
| | POST | /api/v1/auth/login | ❌ |
| | POST | /api/v1/auth/logout | ✅ |
| | POST | /api/v1/auth/refresh | ❌ |
| | GET | /api/v1/auth/me | ✅ |
| | PUT | /api/v1/auth/password | ✅ |
| **Products** | GET | /api/v1/products | ❌ |
| | GET | /api/v1/products/{id} | ❌ |
| | POST | /api/v1/products | ✅ Seller |
| | GET | /api/v1/products/{id}/reviews | ❌ |
| | POST | /api/v1/products/{id}/reviews | ✅ |
| **Categories** | GET | /api/v1/categories | ❌ |
| | POST | /api/v1/categories | ✅ Admin |
| **Cart** | GET | /api/v1/cart | ✅ |
| | POST | /api/v1/cart/items | ✅ |
| | POST | /api/v1/cart/apply-coupon | ✅ |
| **Orders** | POST | /api/v1/orders | ✅ |
| | GET | /api/v1/orders/{id} | ✅ |
| | PUT | /api/v1/orders/{id}/status | ✅ Admin |
| | POST | /api/v1/orders/{id}/cancel | ✅ |
| **Analytics** | GET | /api/v1/analytics/* | ✅ Admin |
