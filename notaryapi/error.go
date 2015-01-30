package notaryapi

import (
	"fmt"
)

const (
	ErrorBadMethod             = 0
	ErrorNotAcceptable         = 1
	ErrorMissingVersionSpec    = 2
	ErrorMalformedVersionSpec  = 3
	ErrorBadVersionSpec        = 4
	ErrorEmptyRequest          = 5
	ErrorBadElementSpec        = 6
	ErrorBadIdentifier         = 7
	ErrorBlockNotFound         = 8
	ErrorEntryNotFound         = 9
	ErrorInternal              = 10
	ErrorJSONMarshal           = 11
	ErrorXMLMarshal            = 12
	ErrorUnsupportedMarshal    = 13
	ErrorJSONUnmarshal         = 14
	ErrorXMLUnmarshal          = 15
	ErrorUnsupportedUnmarshal  = 16
	ErrorBadPOSTData           = 17
	ErrorTemplateError         = 18
	ErrorHTTPNewRequestFailure = 19
	ErrorHTTPDoRequestFailure  = 20
	ErrorHTMLMarshal           = 21
	ErrorVerifySignature       = 22
)

type Error struct {
	APICode     uint
	HTTPCode    int
	Name        string
	Description string
	SupportURL  string
	Message     string
}

func (r *Error) Error() string {
	return fmt.Sprint(r.Name, "\n", r.Description, "\n", r.Message)
}

func CreateError(code uint, message string) *Error {
	r := new(Error)

	r.APICode = code
	r.HTTPCode, r.Name, r.Description, r.SupportURL = retreiveErrorParameters(code)
	r.Message = message

	return r
}

func retreiveErrorParameters(code uint) (int, string, string, string) {
	switch code {
	case ErrorInternal:
		return 500, "Internal", "An internal error occured", ""

	case ErrorJSONMarshal:
		return 500, "JSON Marshal", "An error occured marshalling into JSON", ""

	case ErrorXMLMarshal:
		return 500, "XML Marshal", "An error occured marshalling into XML", ""

	case ErrorUnsupportedMarshal:
		return 500, "Unsupported Marshal", "The server attempted to marshal the data into an unsupported format", ""

	case ErrorBadMethod:
		return 405, "Bad Method", "The specified method cannot be used on the specified resource", ""

	case ErrorNotAcceptable:
		return 406, "Not Acceptable", "The resource cannot be retreived as any of the acceptable types", ""

	case ErrorMissingVersionSpec:
		return 400, "Missing Version Spec", "The API version specifier is missing from the request URL", ""

	case ErrorMalformedVersionSpec:
		return 400, "Malformed Version Spec", "The API version specifier is malformed", ""

	case ErrorBadVersionSpec:
		return 400, "Bad Version Spec", "The API version specifier specifies a bad version", ""

	case ErrorEmptyRequest:
		return 200, "Empty Request", "The request is empty", ""

	case ErrorBadElementSpec:
		return 400, "Bad Element Spec", "The element specifier is bad", ""

	case ErrorBadIdentifier:
		return 400, "Bad Identifier", "The element identifier was malformed", ""

	case ErrorBlockNotFound:
		return 404, "Block Not Found", "The specified block cannot be found", ""

	case ErrorEntryNotFound:
		return 404, "Entry Not Found", "The specified entry cannot be found", ""

	case ErrorJSONUnmarshal:
		return 400, "JSON Unmarshal", "An error occured while unmarshalling from JSON", ""

	case ErrorXMLUnmarshal:
		return 400, "XML Unmarshal", "An error occured while unmarshalling from XML", ""

	case ErrorUnsupportedUnmarshal:
		return 400, "Unsupported Unmarshal", "The data was specified to be in an unsupported format", ""

	case ErrorBadPOSTData:
		return 400, "Bad POST Data", "The body of the POST request is malformed", ""

	case ErrorTemplateError:
		return 500, "Template Error", "A template error occured", ""

	case ErrorHTTPNewRequestFailure:
		return 500, "HTTP Request Failure", "Failed to create an HTTP request", ""

	case ErrorHTTPDoRequestFailure:
		return 500, "HTTP Request Failure", "Error while executing an HTTP request", ""

	case ErrorHTMLMarshal:
		return 500, "HTML Marshal", "An error occured marshalling into HTML", ""
	}

	return 500, "Unknown Error", "An unknown error occured", ""
}
