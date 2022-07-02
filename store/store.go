package store

import (
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
)

type StoreInterface interface {
	Get(key string) (interface{}, error)
	Create(key string, value string) error
	Delete(key string) error
	Update(key string) error
}

type Store struct {
	firestore *FirestoreService
}

func NewStore(driver string) *Store {
	fmt.Println("Creating new store : ", driver)

	return &Store{
		firestore: NewFirestoreService(),
	}
}

func (s *Store) Get(key string) (*firestore.DocumentSnapshot, error) {
	return s.firestore.GetUserRecord(key)
}

func (s *Store) Create(key string, value string) error {
	return s.firestore.AddUserRecord(key, value)
}

func (s *Store) Delete(key, value string) error {
	return s.firestore.DeleteRecord(key, value)
}

//TODO
func (s *Store) Update(key string) error {
	return errors.New("update error")
}
