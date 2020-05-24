all: notoma docs

clean-deps:
	rm -rf requirements-base.txt requirements-dev.txt

deps: clean-deps
	pipenv lock -r > requirements-base.txt
	pipenv lock -rd > requirements-dev.txt

install-deps: deps
	pipenv install
	pipenv install --dev

run-docs: docs
	cd docs && bundle exec jekyll serve

contrib: install-deps
	pipenv run pre-commit install

.PHONY: docs
docs:
	pipenv run notoma-dev docs
	touch docs

# Generate docs from the example Notion database
docs-notion:
	pipenv run notion convert --from "https://www.notion.so/respawn/0b988490f3fc46fcbb6036e652b5a296?v=598cd4e915b94d4da85072f2842117eb" --dest ./docs/

.PHONY: nbexec
nbexec:
	pipenv run nbexec ./notebooks/*.ipynb
	@sleep 1

pypi: dist
	pipenv run twine upload --repository pypi dist/*

pypi-test: dist
	pipenv run twine upload -r testpypi dist/*

dist: clean
	python setup.py sdist bdist_wheel

clean:
	rm -rf dist
