# Twirp Godot

Generate [Twirp](https://github.com/twitchtv/twirp) bindings for [Godot](https://godotengine.org/).

This is experimental at the moment (and also a bit silly), but works for https://github.com/ttab/elephant-api

The only implemented token source is `twirp_ClientCredentials` but it's quite easy to roll your own.

## Installation

Run `go install ./cmd/protoc-gen-twirp_godot` to install the protoc plugin.

## Generating code

This would generate code to communicate with the two services "index" and "repository" and place the code in the folder "twirp" in your repository.

``` shell
protoc \
    --twirp_godot_out=$PATH_TO_GODOT_PROJECT/twirp \
    --proto_path $PROTOBUF_LOCATION/elephant-api \
    index/service.proto repository/service.proto
```

## Usage

``` gdscript
extends Node2D

# Called when the node enters the scene tree for the first time.
func _ready() -> void:
	var creds = twirp_ClientCredentials.new()
	creds.token_endpoint = "https://loginserver.example.com/token"
	creds.client_id = "the-id-of-your-client"
	creds.client_secret = "the-client-secret"
	creds.scope = "doc_read search"
	add_child(creds)

	var documents = elephant_repository_Documents.new(creds, "https://repository.example.com")
	add_child(documents)

    var req = elephant_repository_GetDocumentRequest.from_dictionary({
		"uuid": "958949f3-da6b-4a2e-bb0d-2b94b4fda0b4",
	}, true)

	var res = await documents.Get(req)
	if res.error != "":
		print("failed to get document: %s" % res.error)
	else:
		var resData = elephant_repository_GetDocumentResponse.from_dictionary(res.result)

		print("Document title:", resData.f_document.f_title)
```

The `add_child()` calls are necessary because `HTTPRequest`s must be in the tree to work. This is not needed if the client credentials and services are created as part of the scene.
