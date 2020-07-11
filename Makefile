all: notoma docs

.PHONY: run-docs
run-docs: docs
	cd docs && bundle exec jekyll serve

.PHONY: docs
docs:
	poetry run notoma-dev docs
	touch docs

.PHONY: docs-notion
docs-notion:
	poetry run notoma convert --from "https://www.notion.so/respawn/0b988490f3fc46fcbb6036e652b5a296?v=598cd4e915b94d4da85072f2842117eb" --dest ./docs/

.PHONY: nbexec
nbexec:
	poetry run nbexec ./notebooks/*.ipynb
	@sleep 1

.PHONY: clean
clean:
	rm -rf dist
