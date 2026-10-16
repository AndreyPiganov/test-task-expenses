package service

import "errors"

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrExpenseNotFound = errors.New("expense not found")
	ErrMemberNotFound  = errors.New("family member not found")
)
