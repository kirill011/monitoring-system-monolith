proto_gen:
	protoc --proto_path=proto/api-gateway/users --go_out=proto/api-gateway/users --go_opt=paths=source_relative users.proto
	protoc --proto_path=proto/api-gateway/devices --go_out=proto/api-gateway/devices --go_opt=paths=source_relative devices.proto
	protoc --proto_path=proto/api-gateway/tags --go_out=proto/api-gateway/tags --go_opt=paths=source_relative tags.proto
	protoc --proto_path=proto/api-gateway/messages --go_out=proto/api-gateway/messages --go_opt=paths=source_relative messages.proto




start_service_rebuild:
	docker compose up --build monolith

start_service:
	docker compose up