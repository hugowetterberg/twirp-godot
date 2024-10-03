extends Node
class_name TwirpRequest

var token : String
var server : String
var service : String
var method : String
var _http_request : HTTPRequest

func _rpcResult(result : Dictionary) -> TwirpResponse:
	var res = TwirpResponse.new()
	res.result = result
	return res

func _rpcError(message : String) -> TwirpResponse:
	var res = TwirpResponse.new()
	res.error = message
	return res

func rpcCall(req : Dictionary) -> TwirpResponse:
	_http_request = HTTPRequest.new()
	add_child(_http_request)
	
	var headers = [
		"Content-Type: application/json",
		"Authorization: Bearer %s" % token
	]
	
	var url = "%s/twirp/%s/%s" % [server, service, method]
	var body = JSON.stringify(req)
	
	var error = _http_request.request(url, headers, HTTPClient.METHOD_POST, body)
	if error != OK:
		return _rpcError("make request: %s" % error)
	
	var result : Array = await _http_request.request_completed
	var response_code = result[1]
	var response_body = result[3]
	
	if response_code != 200:
		return _rpcError("error response %d" % response_code)

	var response_object = JSON.parse_string(response_body.get_string_from_utf8())

	_http_request.queue_free()
	_http_request = null

	return _rpcResult(response_object)
