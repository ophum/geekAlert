.PHONY: serve

serve:
	reflex -r '\.go|config.yaml\z' -s -- sh -c 'go run main.go --config config.yaml'
