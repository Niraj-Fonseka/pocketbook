package store

import "fmt"

type StoreInterface interface {
}

type Store struct {
}

func NewStore(driver string) *Store {
	fmt.Println("Creating new store : ", driver)
	return &Store{}
}

func (s *Store) Get() {

}

func (s *Store) Create() {

}

func (s *Store) Delete() {

}

func (s *Store) Update() {

}
