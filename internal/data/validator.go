package data

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func ValidateStruct(s any) (map[string]string, error) {
	err := validate.Struct(s)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return nil, fmt.Errorf("invalid validation error: %w", err)
		}

		validationErrors := err.(validator.ValidationErrors)
		errorsMap := make(map[string]string)
		for _, fieldErr := range validationErrors {
			errorsMap[fieldErr.Field()] = fmt.Sprintf("failed on '%s' tag", fieldErr.Tag())
		}
		return errorsMap, fmt.Errorf("validation errors")
	}
	return nil, nil
}
