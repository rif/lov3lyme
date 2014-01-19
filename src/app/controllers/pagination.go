package controllers

import (
	"fmt"
	"net/url"
	"strconv"
)

type Pagination struct {
	Max     int
	PerPage int
	Current int
}

const (
	pagination_perpage = "perpage"
	pagination_current = "page"
)

func NewPagination(max int, query url.Values) (p *Pagination) {
	p = &Pagination{
		Max:     max,
		PerPage: 10,
		Current: 1,
	}
	per_page := query.Get(pagination_perpage)
	if n, err := strconv.ParseInt(per_page, 10, 32); err == nil {
		p.PerPage = int(n)
	}
	current := query.Get(pagination_current)
	if n, err := strconv.ParseInt(current, 10, 32); err == nil {
		p.Current = int(n)
	}

	//set some bounds on the page and number of items per page
	if p.Current < 1 {
		p.Current = 1
	}
	if p.PerPage < 1 {
		p.PerPage = 1
	}
	return
}

func (p *Pagination) PageLink(n int) string {
	return "?" + url.Values{
		pagination_perpage: {fmt.Sprint(p.PerPage)},
		pagination_current: {fmt.Sprint(n)},
	}.Encode()
}

func (p *Pagination) Last() int {
	return ((p.Max - 1) / p.PerPage) + 1
}

func (p *Pagination) First() int {
	return 1
}

func (p *Pagination) Prev() int {
	if p.Current <= p.First() {
		return p.Current
	}
	return p.Current - 1
}

func (p *Pagination) Next() int {
	if p.Current >= p.Last() {
		return p.Last()
	}
	return p.Current + 1
}

func (p *Pagination) BeforePages() (r []int) {
	low := p.Current - 5
	if low < p.First() {
		low = p.First()
	}
	for low < p.Current {
		r = append(r, low)
		low += 1
	}
	return
}

func (p *Pagination) AfterPages() (r []int) {
	top := p.Current + 5
	if top > p.Last() {
		top = p.Last()
	}
	for high := p.Current + 1; high <= top; high++ {
		r = append(r, high)
	}
	return
}

func (p *Pagination) Range() (low, hi int) {
	low = p.PerPage * (p.Current - 1)
	hi = (p.PerPage * p.Current) - 1

	//dont go off the top end
	if hi > p.Max {
		hi = p.Max
	}
	//dont go off negative
	if low < 0 {
		low = 0
	}
	//make sure low < hi!
	if low > hi {
		hi = low
	}

	return
}

func (p *Pagination) Show() bool {
	return p.Max > p.PerPage
}
