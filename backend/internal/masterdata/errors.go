package masterdata

import "fmt"

type ValidationError struct{ Fields FieldErrors }

func (e ValidationError) Error() string { return "validation failed" }

type NotFoundError struct{ Resource string }

func (e NotFoundError) Error() string { return fmt.Sprintf("%s not found", e.Resource) }

type ConflictError struct{ Fields FieldErrors }

func (e ConflictError) Error() string { return "record conflicts with existing data" }
