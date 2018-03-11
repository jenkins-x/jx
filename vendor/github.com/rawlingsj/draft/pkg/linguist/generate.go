package linguist

//go:generate cp data/linguist/lib/linguist/languages.yml data/
//go:generate cp data/linguist/lib/linguist/documentation.yml data/
//go:generate cp data/linguist/lib/linguist/vendor.yml data/
//go:generate go run generate_static.go data/languages.yml data/vendor.yml data/documentation.yml
