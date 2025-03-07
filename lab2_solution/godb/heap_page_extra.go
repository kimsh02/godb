package godb

// Returns the page number of the page.
func (p *heapPage) PageNo() int {
	//<strip lab5>
	return p.pageNo
	//</strip>
}
