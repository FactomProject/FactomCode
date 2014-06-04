package main

import (

)

const (
	errorBadMethod = 0
	errorNotAcceptable = 1
	errorMissingVersionSpec = 2
	errorMalformedVersionSpec = 3
	errorBadVersionSpec = 4
	errorEmptyRequest = 5
	errorBadElementSpec = 6
	errorBadIdentifier = 7
	errorBlockNotFound = 8
	errorEntryNotFound = 9
	errorInternal = 10
	errorJSONMarshal = 11
	errorXMLMarshal = 12
	errorUnsupportedMarshal = 13
	errorJSONUnmarshal = 14
	errorXMLUnmarshal = 15
	errorUnsupportedUnmarshal = 16
	errorBadPOSTData = 17
)

type restError struct {
	APICode				uint
	HTTPCode			int
	Name				string
	Description			string
	SupportURL			string
	Message				string
}

func (r *restError) Error() string {
	return r.Description
}

func createError(code uint, message string) *restError {
	r := new(restError)
	
	r.APICode = code
	r.HTTPCode, r.Name, r.Description, r.SupportURL = retreiveErrorParameters(code)
	r.Message = message
	
	return r
}

func retreiveErrorParameters(code uint) (int, string, string, string) {
	switch code {
	case errorInternal:
		return 500, "Internal", "An internal error occured", ""
		
	case errorJSONMarshal:
		return 500, "JSON Marshal", "An error occured marshalling into JSON", ""
		
	case errorXMLMarshal:
		return 500, "XML Marshal", "An error occured marshalling into XML", ""
		
	case errorUnsupportedMarshal:
		return 500, "Unsupported Marshal", "The server attempted to marshal the data into an unsupported format", ""
	
	case errorBadMethod:
		return 405, "Bad Method", "The specified method cannot be used on the specified resource", ""
		
	case errorNotAcceptable:
		return 406, "Not Acceptable", "The resource cannot be retreived as any of the acceptable types", ""
		
	case errorMissingVersionSpec:
		return 400, "Missing Version Spec", "The API version specifier is missing from the request URL", ""
		
	case errorMalformedVersionSpec:
		return 400, "Malformed Version Spec", "The API version specifier is malformed", ""
		
	case errorBadVersionSpec:
		return 400, "Bad Version Spec", "The API version specifier specifies a bad version", ""
		
	case errorEmptyRequest:
		return 200, "Empty Request", "The request is empty", ""
		
	case errorBadElementSpec:
		return 400, "Bad Element Spec", "The element specifier is bad", ""
		
	case errorBadIdentifier:
		return 400, "Bad Identifier", "The element identifier was malformed", ""
		
	case errorBlockNotFound:
		return 404, "Block Not Found", "The specified block cannot be found", ""
		
	case errorEntryNotFound:
		return 404, "Entry Not Found", "The specified entry cannot be found", ""
		
	case errorJSONUnmarshal:
		return 400, "JSON Unmarshal", "An error occured while unmarshalling from JSON", ""
		
	case errorXMLUnmarshal:
		return 400, "XML Unmarshal", "An error occured while unmarshalling from XML", ""
		
	case errorUnsupportedUnmarshal:
		return 400, "Unsupported Unmarshal", "The data was specified to be in an unsupported format", ""
		
	case errorBadPOSTData:
		return 400, "Bad POST Data", "The body of the POST request is malformed", ""
	}
	
	return 500, "Unknown Error", "An unknown error occured", ""
}