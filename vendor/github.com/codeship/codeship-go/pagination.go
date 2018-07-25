package codeship

import (
	"net/url"
	"strconv"
)

type pagination struct {
	Total   int `json:"total,omitempty"`
	PerPage int `json:"per_page,omitempty"`
	Page    int `json:"page,omitempty"`
}

// PaginationOption is a functional option for providing pagination options
type PaginationOption func(o *paginationOption)

type paginationOption struct {
	perPage int
	page    int
}

// Page sets the page of results to be returned in the response
func Page(page int) PaginationOption {
	return func(o *paginationOption) {
		o.page = page
	}
}

// PerPage sets the number of results to be returned per page in the response
func PerPage(perPage int) PaginationOption {
	return func(o *paginationOption) {
		o.perPage = perPage
	}
}

func paginate(path string, opts ...PaginationOption) (string, error) {
	if len(opts) == 0 {
		return path, nil
	}

	opt := &paginationOption{}

	for _, f := range opts {
		f(opt)
	}

	u, err := url.Parse(path)
	if err != nil {
		return path, err
	}

	q := u.Query()
	if opt.page > 0 {
		q.Add("page", strconv.Itoa(opt.page))
	}
	if opt.perPage > 0 {
		q.Add("per_page", strconv.Itoa(opt.perPage))
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Links contain links for pagination purposes
//
// Codeship API docs: https://apidocs.codeship.com/v2/introduction/pagination
type Links struct {
	Next     string
	Previous string
	Last     string
	First    string
}

// NextPage returns the page number of the next page
func (l Links) NextPage() (int, error) {
	if l.Next == "" {
		return 0, nil
	}
	return pageForURL(l.Next)
}

// PreviousPage returns the page number of the previous page
func (l Links) PreviousPage() (int, error) {
	if l.Previous == "" {
		return 0, nil
	}
	return pageForURL(l.Previous)
}

// CurrentPage returns the page number of the current page
func (l Links) CurrentPage() (int, error) {
	switch {
	case l.Previous == "" && l.Next != "":
		return 1, nil
	case l.Previous != "":
		prevPage, err := pageForURL(l.Previous)
		if err != nil {
			return 0, err
		}

		return prevPage + 1, nil
	}
	return 0, nil
}

// LastPage returns the page number of the last page
func (l Links) LastPage() (int, error) {
	if l.Last == "" {
		return l.CurrentPage()
	}
	return pageForURL(l.Last)
}

// IsLastPage returns true if the current page is the last
func (l Links) IsLastPage() bool {
	return l.Last == ""
}

func pageForURL(path string) (int, error) {
	u, err := url.ParseRequestURI(path)
	if err != nil {
		return 0, err
	}

	pageStr := u.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return 0, err
	}

	return page, nil
}
