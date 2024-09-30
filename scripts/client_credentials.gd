extends Node
class_name twirp_ClientCredentials

@export var token_endpoint : String
@export var client_id : String
@export var client_secret : String
@export var scope : String

var token : String
var in_progress : bool
var expiryTimer : Timer = Timer.new()

var _http_request : HTTPRequest

signal got_token(token: String)
signal token_expired

func _ready() -> void:
	expiryTimer.timeout.connect(_token_expired)
	expiryTimer.one_shot = true
	expiryTimer.autostart = false
	add_child(expiryTimer)

func _token_expired():
	token = ""
	token_expired.emit()

func get_token() -> String:
	if token != "":
		return token

	if in_progress:
		await got_token
		return token
	
	in_progress = true
	
	_http_request = HTTPRequest.new()
	add_child(_http_request)
	
	var headers = [
		"Content-Type: application/x-www-form-urlencoded"
	]
	
	var body_parts = [
		"client_id=%s" % client_id,
		"client_secret=%s" % client_secret,
		"scope=%s" % scope,
		"grant_type=client_credentials"
	]
	
	var body = "&".join(body_parts)
	
	var error = _http_request.request(token_endpoint, headers, HTTPClient.METHOD_POST, body)
	if error != OK:
		push_error("make token request: %s", error)
		
	var result : Array = await _http_request.request_completed
	var response_code = result[1]
	var response_body = result[3]
	
	if response_code != 200:
		push_error("error response %d when authenticating" % response_code)
		return ""

	var response_object = JSON.parse_string(response_body.get_string_from_utf8())

	_http_request.queue_free()
	_http_request = null

	token = response_object["access_token"]
	var expires_in : float = response_object["expires_in"]

	expiryTimer.start(expires_in * .9)
	
	got_token.emit(token)
	
	return token
