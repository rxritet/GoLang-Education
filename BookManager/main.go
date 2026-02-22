package main

import (
	"fmt"
	"log"
	"net/http"
	"thirdproject/handlers"
	"thirdproject/models"
)

func main() {
	// Создаём хранилище и обработчики
	store := models.NewStore()
	h := handlers.New(store)

	mux := http.NewServeMux()

	// Статические файлы (index.html и т.д.)
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	// API маршруты:
	//   GET    /api/books        — список всех книг
	//   POST   /api/books        — создать книгу
	//   GET    /api/books/{id}   — получить книгу по ID
	//   PUT    /api/books/{id}   — обновить книгу по ID
	//   DELETE /api/books/{id}   — удалить книгу по ID
	mux.HandleFunc("/api/books", h.BooksRouter)
	mux.HandleFunc("/api/books/", h.BooksRouter)

	addr := ":8080"
	fmt.Printf("Сервер запущен: http://localhost%s\n", addr)
	fmt.Println("Примеры запросов:")
	fmt.Println("  GET    http://localhost:8080/api/books")
	fmt.Println("  GET    http://localhost:8080/api/books/1")
	fmt.Println("  POST   http://localhost:8080/api/books   (body: JSON)")
	fmt.Println("  PUT    http://localhost:8080/api/books/1 (body: JSON)")
	fmt.Println("  DELETE http://localhost:8080/api/books/1")

	log.Fatal(http.ListenAndServe(addr, mux))
}
