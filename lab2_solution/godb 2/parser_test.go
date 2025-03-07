package godb

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"text/tabwriter"
	"time"
)

func MakeTestDatabase(bufferPoolSize int, catalog string) (*BufferPool, *Catalog, error) {
	bp, err := NewBufferPool(bufferPoolSize)
	if err != nil {
		return nil, nil, err
	}

	// load the catalog so we know which tables to remove
	c := NewCatalog(catalog, bp, "./")
	if err := c.parseCatalogFile(); err != nil {
		return nil, nil, err
	}
	for tableName := range c.tableMap {
		os.Remove(c.tableNameToFile(tableName))
	}

	// reload the catalog to reopen the table files
	c = NewCatalog(catalog, bp, "./")
	if err := c.parseCatalogFile(); err != nil {
		return nil, nil, err
	}

	os.Remove("test.log")
	lf, err := NewLogFile("test.log", bp, c)
	if err != nil {
		return nil, nil, err
	}

	if err := bp.Recover(lf); err != nil {
		return nil, nil, err
	}

	return bp, c, nil
}

func RecoverTestDatabase(bufferPoolSize int, catalog string) (*BufferPool, *Catalog, error) {
	bp, err := NewBufferPool(bufferPoolSize)
	if err != nil {
		return nil, nil, err
	}

	c := NewCatalog(catalog, bp, "./")
	if err := c.parseCatalogFile(); err != nil {
		return nil, nil, err
	}
	lf, err := NewLogFile("test.log", bp, c)
	if err != nil {
		return nil, nil, err
	}

	if err := bp.Recover(lf); err != nil {
		return nil, nil, err
	}

	return bp, c, nil
}

func MakeParserTestDatabase(bufferPoolSize int) (*BufferPool, *Catalog, error) {
	os.Remove("t2.dat")
	os.Remove("t.dat")

	bp, c, err := MakeTestDatabase(bufferPoolSize, "catalog.txt")
	if err != nil {
		return nil, nil, err
	}

	hf, err := c.GetTable("t")
	if err != nil {
		return nil, nil, err
	}
	hf2, err := c.GetTable("t2")
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open("testdb.txt")
	if err != nil {
		return nil, nil, err
	}
	err = hf.(*HeapFile).LoadFromCSV(f, true, ",", false)
	if err != nil {
		return nil, nil, err
	}

	f, err = os.Open("testdb.txt")
	if err != nil {
		return nil, nil, err
	}
	err = hf2.(*HeapFile).LoadFromCSV(f, true, ",", false)
	if err != nil {
		return nil, nil, err
	}

	if err := c.ComputeTableStats(); err != nil {
		return nil, nil, err
	}

	return bp, c, nil
}

func TestParseBadQueries(t *testing.T) {
	var badQueries []string = []string{
		"select name from t join t2 on t.name = t2.name",  //name is ambiguous
		"select age from (select age age2 from t) x",      //age is not in select list
		"select age from (select age age2 from t)",        //subquery is not named
		"select age from t join t2 on name = name",        //name is ambiguous
		"select age from t join t on name = name",         //can't join table twice without alias
		"select age from t join t t2 on t2.name = name",   //name is unqualified
		"select age from t join t t2 on t2.name = t.name", //age is unqualified

	}

	_, c, err := MakeParserTestDatabase(10)
	if err != nil {
		t.Fatalf("failed to create test database, %s", err.Error())
	}

	for _, sql := range badQueries {
		_, _, err := Parse(c, sql)
		if err == nil {
			t.Errorf("query %s parsed, expected it to fail", sql)
		}
	}
}

// Return a string representing a table of tuples.
func PrettyTable(ts []*Tuple) string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)

	if len(ts) > 0 {
		for _, f := range ts[0].Desc.Fields {
			if f.TableQualifier != "" {
				fmt.Fprintf(w, "%s.", f.TableQualifier)
			}
			fmt.Fprintf(w, "%s\t", f.Fname)

		}
		fmt.Fprint(w, "\n")
	}

	for _, t := range ts {
		for _, f := range t.Fields {
			switch f := f.(type) {
			case IntField:
				fmt.Fprintf(w, "%d\t", f.Value)
			case StringField:
				fmt.Fprintf(w, "%s\t", f.Value)
			}
		}
		fmt.Fprint(w, "\n")
	}
	w.Flush()

	return buf.String()
}

func TestParse(t *testing.T) {
	skip := []int{14}
	queries := []Query{
		{SQL: "select * from t,(select name name2 from t) t3 where name=t3.name2", Ordered: false},
		{SQL: "select * from t,(select name name2 from t) t3 where name=name2", Ordered: false},
		{SQL: "select * from t join (select name name2 from t) t3 on name=name2", Ordered: false},
		{SQL: "select sum(age + 10) , sum(age) from t", Ordered: false},
		{SQL: "select min(age) + max(age) from t", Ordered: false},
		{SQL: "select * from t limit 1", Ordered: false},
		{SQL: "select * from t limit 1+2", Ordered: false},
		{SQL: "select sum(a) from (select 1+2 a,2, name from t) t2", Ordered: false},
		{SQL: "select t.name, t.age from t join t2 on t.name = t2.name, t2 as t3 where t.age < 50 and t3.age = t.age order by t.age asc, t.name asc", Ordered: true},
		{SQL: "select sq(sq(5)) from t", Ordered: false},
		{SQL: "select getsubstr(name,1,2), sum(t.age + 10) s1, sum(t.age) s2, count(*) from t group by getsubstr(name,1,2)", Ordered: false},
		{SQL: "select sum(t.age + 10) s1, sum(t.age) s2 from t", Ordered: false},
		{SQL: "select sum(num) from (select sum(age) num from t group by name) v", Ordered: false},
		{SQL: "select t.name, t2.name from t2 join t on getsubstr(t.name,1,1) = getsubstr(t2.name,1,1)", Ordered: false},
		{SQL: "select t2.name from t2 join t on t.name = t2.name where t.name = 'sam'", Ordered: false},
		{SQL: "select t.name, t.age from t join t2 on t.name = t2.name where t.age > 50", Ordered: false},
		{SQL: "select name, age from t where age + 1 < age + 2", Ordered: false},
		{SQL: "select age from t where t.age + 10 < 25 + 10", Ordered: false},
		{SQL: "select count(*) from t group by getsubstr(name, 1, 2)", Ordered: false},
		{SQL: "select getsubstr(name, 1, 2), name, age from t order by getsubstr(name, 1, 2)", Ordered: true},
		{SQL: "select 1, name from t", Ordered: false},
		{SQL: "select age, name from t", Ordered: false},
		{SQL: "select count(*) countages from (select t.name, sum(age) totage from t group by t.name) x", Ordered: false},
		{SQL: "select cnt from (select count(*) cnt from (select t.name, sum(t.age) agesum from t,t2 where t.name = t2.name group by t.name) t3 where agesum > 20) t4", Ordered: false},
		{SQL: "select name, name n2 from (select * from t) x", Ordered: false},
		{SQL: "select t.name, sum(age) totage from t group by t.name", Ordered: false},
		{SQL: "select t.name, t.age from t join t2 on t.name = t2.name where t.age < 50", Ordered: false},
		{SQL: "select foo.name from t foo join (select t2.name from t2 join t on t.name = t2.name where t.name = 'sam') q on q.name = foo.name order by foo.name", Ordered: true},
		{SQL: "select y.age from t y join t x on x.name = y.name", Ordered: false},
		{SQL: "select name from (select x.name from (select t.name from t) x)y order by name asc", Ordered: true},
		{SQL: "select name, age from (select t.name, t.age from t join t2 on t.name = t2.name join t t3  on t.name = t3.name where t.age > 50 order by t.name asc) q where age > 60", Ordered: false},
		{SQL: "select a, b from (select sum(bar) a, count(foo) b from (select t.name foo, t.age bar from t join t2 on t.name = t2.name)x )y", Ordered: false},
		{SQL: "select age, count(*) from t group by age", Ordered: false},
	}
	save := false        //set save to true to save the output of the current test run as the correct answer
	printOutput := false //print the result set during testing

	bp, c, err := MakeParserTestDatabase(10)
	if err != nil {
		t.Fatalf("failed to create test database, %s", err.Error())
	}

	qNo := 0
	for _, query := range queries {
		tid := NewTID()
		bp.BeginTransaction(tid)
		qNo++

		shouldSkip := false
		for _, s := range skip {
			if s == qNo {
				shouldSkip = true
			}
		}
		if shouldSkip {
			continue
		}

		qType, plan, err := Parse(c, query.SQL)
		if err != nil {
			t.Fatalf("failed to parse, q%d=%s, %s", qNo, query.SQL, err.Error())
		}
		if qType != IteratorType {
			continue
		}

		var outfile *HeapFile
		var resultSet []*Tuple
		fname := fmt.Sprintf("savedresults/q%d-result.dat", qNo)

		if save {
			os.Remove(fname)
			outfile, _ = NewHeapFile(fname, plan.Descriptor(), bp)
		} else {
			fname := fmt.Sprintf("savedresults/q%d-result.dat", qNo)
			outfile, _ = NewHeapFile(fname, plan.Descriptor(), bp)
			resultIter, err := outfile.Iterator(tid)
			if err != nil {
				t.Fatalf("%s", err.Error())
			}
			for {
				tup, err := resultIter()
				if err != nil {
					t.Fatalf("%s", err.Error())
				}
				if tup == nil {
					break
				}
				resultSet = append(resultSet, tup)
			}
		}

		if printOutput || save {
			fmt.Printf("Doing %s\n", query.SQL)
			start := time.Now()
			iter, err := plan.Iterator(tid)
			if err != nil {
				t.Errorf("%s", err.Error())
				return
			}
			nresults := 0
			fmt.Printf("%s\n", plan.Descriptor().HeaderString(true))
			for {
				tup, err := iter()
				if err != nil {
					t.Fatalf("%s", err.Error())
				}
				if tup == nil {
					break
				}
				fmt.Printf("%s\n", tup.PrettyPrintString(true))
				nresults++
				if save {
					insertTupleForTest(t, outfile, tup, tid)
				}
			}
			end := time.Now()
			fmt.Printf("(%d results) %v\n\n", nresults, end.Sub(start))
		}
		if save {
			bp.FlushAllPages()
			outfile.bufPool.CommitTransaction(tid)
		} else {
			iter, err := plan.Iterator(tid)
			if err != nil {
				t.Fatalf("%s", err.Error())
			}
			if query.Ordered {
				err = CheckIfOutputMatches(iter, resultSet)
			} else {
				err = CheckIfOutputMatchesUnordered(iter, resultSet)
			}
			if err != nil {
				t.Errorf("query '%s' did not match expected result set: %v", query.SQL, err)
				verbose := true
				if verbose {
					t.Logf("Expected: \n")
					t.Logf("%s\n", PrettyTable(resultSet))
				}
			}
		}
	}
}
