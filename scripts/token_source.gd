extends Node
class_name TwirpTokenSource

signal got_token(token: String)
signal token_expired

func get_token() -> String:
	var res = await got_token
	return res[0]
