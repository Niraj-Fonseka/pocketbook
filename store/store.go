package store

import (
	"errors"
	"fmt"
)

type StoreInterface interface {
	Get(key string) (interface{}, error)
	Create(key string, value string) error
	Delete(key string) error
	Update(key string) error
}

type Store struct {
}

func NewStore(driver string) *Store {
	fmt.Println("Creating new store : ", driver)
	return &Store{}
}

func (s *Store) Get(key string) (interface{}, error) {
	var a interface{}

	return a, errors.New("new error")
}

func (s *Store) Create(key string, value string) error {
	return errors.New("new error")
}

func (s *Store) Delete(key string) error {
	return errors.New("delete error")
}

func (s *Store) Update(key string) error {
	return errors.New("update error")
}
