package simulation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"foundry-tx-simulator/backend/internal/model"
	"foundry-tx-simulator/backend/internal/solidity"
)

var simulateRequestValidator = newSimulateRequestValidator()

func newSimulateRequestValidator() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterTagNameFunc(jsonFieldName)
	_ = validate.RegisterValidation("eth_address", validateAddress)
	_ = validate.RegisterValidation("hex_bytes", validateHexBytes)
	_ = validate.RegisterValidation("notblank", validateNotBlank)
	return validate
}

func validateSimulateRequest(req *model.SimulateRequest) error {
	if err := simulateRequestValidator.Struct(req); err != nil {
		return formatValidationError(err)
	}
	return nil
}

func jsonFieldName(field reflect.StructField) string {
	name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
	if name == "" || name == "-" {
		return field.Name
	}
	return name
}

func validateAddress(level validator.FieldLevel) bool {
	return solidity.ValidateAddress("", level.Field().String()) == nil
}

func validateHexBytes(level validator.FieldLevel) bool {
	_, err := solidity.NormalizeBytes("", level.Field().String())
	return err == nil
}

func validateNotBlank(level validator.FieldLevel) bool {
	return strings.TrimSpace(level.Field().String()) != ""
}

func formatValidationError(err error) error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok || len(validationErrors) == 0 {
		return err
	}

	fieldError := validationErrors[0]
	field := validationFieldPath(fieldError)
	switch fieldError.Tag() {
	case "required", "notblank":
		return fmt.Errorf("%s is required", field)
	case "eth_address":
		return fmt.Errorf("%s must be a 20-byte hex address", field)
	case "hex_bytes":
		return fmt.Errorf("%s must be even-length hex bytes", field)
	default:
		return fmt.Errorf("%s is invalid", field)
	}
}

func validationFieldPath(fieldError validator.FieldError) string {
	namespace := fieldError.Namespace()
	namespace = strings.TrimPrefix(namespace, "SimulateRequest.")
	if namespace == "" {
		return fieldError.Field()
	}
	return namespace
}
