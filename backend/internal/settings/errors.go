package settings

import "fmt"

type ValidationError struct{ Fields FieldErrors }

func (ValidationError) Error() string { return "validation failed" }

type ConflictError struct{ Fields FieldErrors }

func (ConflictError) Error() string { return "settings conflict" }

type NotFoundError struct{ Resource string }

func (e NotFoundError) Error() string { return fmt.Sprintf("%s not found", e.Resource) }
