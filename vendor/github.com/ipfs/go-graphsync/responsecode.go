package graphsync

import "fmt"

// ResponseStatusCode is a status returned for a GraphSync Request.
type ResponseStatusCode int32

// GraphSync Response Status Codes
const (
	// Informational Response Codes (partial)

	// RequestAcknowledged means the request was received and is being worked on.
	RequestAcknowledged = ResponseStatusCode(10)
	// AdditionalPeers means additional peers were found that may be able
	// to satisfy the request and contained in the extra block of the response.
	AdditionalPeers = ResponseStatusCode(11)
	// NotEnoughGas means fulfilling this request requires payment.
	NotEnoughGas = ResponseStatusCode(12)
	// OtherProtocol means a different type of response than GraphSync is
	// contained in extra.
	OtherProtocol = ResponseStatusCode(13)
	// PartialResponse may include blocks and metadata about the in progress response
	// in extra.
	PartialResponse = ResponseStatusCode(14)
	// RequestPaused indicates a request is paused and will not send any more data
	// until unpaused
	RequestPaused = ResponseStatusCode(15)

	// Success Response Codes (request terminated)

	// RequestCompletedFull means the entire fulfillment of the GraphSync request
	// was sent back.
	RequestCompletedFull = ResponseStatusCode(20)
	// RequestCompletedPartial means the response is completed, and part of the
	// GraphSync request was sent back, but not the complete request.
	RequestCompletedPartial = ResponseStatusCode(21)

	// Error Response Codes (request terminated)

	// RequestRejected means the node did not accept the incoming request.
	RequestRejected = ResponseStatusCode(30)
	// RequestFailedBusy means the node is too busy, try again later. Backoff may
	// be contained in extra.
	RequestFailedBusy = ResponseStatusCode(31)
	// RequestFailedUnknown means the request failed for an unspecified reason. May
	// contain data about why in extra.
	RequestFailedUnknown = ResponseStatusCode(32)
	// RequestFailedLegal means the request failed for legal reasons.
	RequestFailedLegal = ResponseStatusCode(33)
	// RequestFailedContentNotFound means the respondent does not have the content.
	RequestFailedContentNotFound = ResponseStatusCode(34)
	// RequestCancelled means the responder was processing the request but decided to top, for whatever reason
	RequestCancelled = ResponseStatusCode(35)
)

func (c ResponseStatusCode) String() string {
	str, ok := ResponseCodeToName[c]
	if ok {
		return str
	}
	return fmt.Sprintf("UnknownResponseCode %d", c)
}

var ResponseCodeToName = map[ResponseStatusCode]string{
	RequestAcknowledged:          "RequestAcknowledged",
	AdditionalPeers:              "AdditionalPeers",
	NotEnoughGas:                 "NotEnoughGas",
	OtherProtocol:                "OtherProtocol",
	PartialResponse:              "PartialResponse",
	RequestPaused:                "RequestPaused",
	RequestCompletedFull:         "RequestCompletedFull",
	RequestCompletedPartial:      "RequestCompletedPartial",
	RequestRejected:              "RequestRejected",
	RequestFailedBusy:            "RequestFailedBusy",
	RequestFailedUnknown:         "RequestFailedUnknown",
	RequestFailedLegal:           "RequestFailedLegal",
	RequestFailedContentNotFound: "RequestFailedContentNotFound",
	RequestCancelled:             "RequestCancelled",
}

// AsError generates an error from the status code for a failing status
func (c ResponseStatusCode) AsError() error {
	if c.IsSuccess() {
		return nil
	}
	switch c {
	case RequestFailedBusy:
		return RequestFailedBusyErr{}
	case RequestFailedContentNotFound:
		return RequestFailedContentNotFoundErr{}
	case RequestFailedLegal:
		return RequestFailedLegalErr{}
	case RequestFailedUnknown:
		return RequestFailedUnknownErr{}
	case RequestCancelled:
		return RequestCancelledErr{}
	default:
		return fmt.Errorf("unknown response status code: %d", c)
	}
}

// IsSuccess returns true if the response code indicates the
// request terminated successfully.
func (c ResponseStatusCode) IsSuccess() bool {
	return c == RequestCompletedFull || c == RequestCompletedPartial
}

// IsFailure returns true if the response code indicates the
// request terminated in failure.
func (c ResponseStatusCode) IsFailure() bool {
	return c == RequestFailedBusy ||
		c == RequestFailedContentNotFound ||
		c == RequestFailedLegal ||
		c == RequestFailedUnknown ||
		c == RequestCancelled ||
		c == RequestRejected
}

// IsTerminal returns true if the response code signals
// the end of the request
func (c ResponseStatusCode) IsTerminal() bool {
	return c.IsSuccess() || c.IsFailure()
}
