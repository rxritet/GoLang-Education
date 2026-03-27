package models

import "sync"

// Book представляет книгу в нашем хранилище
type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

// Store — потокобезопасное in-memory хранилище книг
type Store struct {
	mu     sync.RWMutex
	books  map[int]Book
	nextID int
}

// NewStore создаёт новое хранилище с тестовыми данными
func NewStore() *Store {
	s := &Store{
		books:  make(map[int]Book),
		nextID: 1,
	}

	// Добавим несколько книг по умолчанию
	s.books[1] = Book{ID: 1, Title: "The Go Programming Language", Author: "Alan A. A. Donovan", Year: 2015}
	s.books[2] = Book{ID: 2, Title: "Clean Code", Author: "Robert C. Martin", Year: 2008}
	s.books[3] = Book{ID: 3, Title: "The Pragmatic Programmer", Author: "Andrew Hunt", Year: 1999}
	s.nextID = 4

	return s
}

// GetAll возвращает все книги
func (s *Store) GetAll() []Book {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Book, 0, len(s.books))
	for _, b := range s.books {
		list = append(list, b)
	}
	return list
}

// GetByID возвращает книгу по ID, или false если не найдена
func (s *Store) GetByID(id int) (Book, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.books[id]
	return b, ok
}

// Create добавляет новую книгу и возвращает её с присвоенным ID
func (s *Store) Create(b Book) Book {
	s.mu.Lock()
	defer s.mu.Unlock()

	b.ID = s.nextID
	s.nextID++
	s.books[b.ID] = b
	return b
}

// Update обновляет существующую книгу, возвращает false если не найдена
func (s *Store) Update(id int, updated Book) (Book, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.books[id]; !ok {
		return Book{}, false
	}
	updated.ID = id
	s.books[id] = updated
	return updated, true
}

// Delete удаляет книгу по ID, возвращает false если не найдена
func (s *Store) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.books[id]; !ok {
		return false
	}
	delete(s.books, id)
	return true
}
