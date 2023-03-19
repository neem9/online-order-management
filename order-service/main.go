package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Order struct {
	ID            int       `json:"id"`
	ProductID     int       `json:"product_id"`
	ProductPrice  float64   `json:"product_price"`
	ProductQty    int       `json:"product_qty"`
	Discount      float64   `json:"discount"`
	OrderValue    float64   `json:"order_value"`
	OrderStatus   string    `json:"order_status"`
	DispatchDate  time.Time `json:"dispatch_date"`
	OrderDateTime time.Time `json:"order_date_time"`
}

var (
	orders  = make(map[int]*Order)
	orderID = 1
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/orders", getOrdersHandler).Methods(http.MethodGet)
	r.HandleFunc("/orders", createOrderHandler).Methods(http.MethodPost)
	r.HandleFunc("/orders/{id}", updateOrderHandler).Methods(http.MethodPatch)

	fmt.Println("Order Server is listening on :8001...")
	err := http.ListenAndServe(":8001", r)
	if err != nil {
		panic(err)
	}
}

func getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ordersList := make([]*Order, 0, len(orders))
	for _, order := range orders {
		ordersList = append(ordersList, order)
	}
	json.NewEncoder(w).Encode(ordersList)
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var order Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	order.OrderDateTime = time.Now()
	order.Discount = 0
	order.OrderValue = order.ProductPrice * float64(order.ProductQty)

	if order.OrderStatus == "" {
		order.OrderStatus = "Placed"
	} else if order.OrderStatus != "Placed" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid order status: %s", order.OrderStatus)
		return
	}

	if order.ProductQty > 10 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Quantity exceeds maximum allowed: %d", order.ProductQty)
		return
	}

	orders[orderID] = &order
	order.ID = orderID
	orderID++
	json.NewEncoder(w).Encode(order)
}

func updateOrderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid order ID: %s", idStr)
		return
	}

	order, ok := orders[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Order not found: %d", id)
		return
	}

	var update Order
	err = json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	if update.OrderStatus != "" {
		if update.OrderStatus != "Dispatched" && update.OrderStatus != "Completed" && update.OrderStatus != "Cancelled" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Invalid order status: %s", update.OrderStatus)
			return
		}
		order.OrderStatus = update.OrderStatus
	}

	if order.OrderStatus == "Dispatched" {
		if update.DispatchDate.IsZero() {
			update.DispatchDate = time.Now()
		}
		order.DispatchDate = update.DispatchDate
	}

	orders[id] = order

	json.NewEncoder(w).Encode(order)
}
