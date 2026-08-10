package executor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func decodeStrictPayload[T any](
	rawPayload json.RawMessage,
) (T, error) {
	var payload T

	decoder := json.NewDecoder(
		bytes.NewReader(rawPayload),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		return payload, fmt.Errorf(
			"%w: %v",
			ErrInvalidHandlerPayload,
			err,
		)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(
		err,
		io.EOF,
	) {
		return payload, fmt.Errorf(
			"%w: payload must contain one JSON object",
			ErrInvalidHandlerPayload,
		)
	}

	return payload, nil
}
