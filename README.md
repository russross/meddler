sqlscan
=======

A small toolkit to automatically scan sql results into structs and
generate insert and update statements based on structs.

Package docs: http://godoc.org/github.com/russross/sqlscan

This is currently designed for Sqlite, MySQL, and PostgreSQL, though
it has not been tested on PostgreSQL. If you test it, please let me
know if it works or not and I will update this README.

To use with PostgreSQL, set the following:

    sqlscan.Quote = "\""
	sqlscan.Placeholder = "$1"
	sqlscan.PostgreSQL = true


High-level functions
--------------------

sqlscan does not create or alter tables. It just provides a little
glue to make it easier to read and write structs as SQL rows. Start
by annotating a struct:

``` go
type Person struct {
    ID      int       `sqlscan:"id,pk"`
    Name    string    `sqlscan:"name"`
    Created time.Time `sqlscan:"created,localtime"`
    Closed  time.Time `sqlscan:",localtimez"`
}
```

Notes about this example:

*   Non-public fields are ignored by sqlscan
*   If the optional tag is provided, the first field is the database
    column name. Note that "Closed" does not provide a column name,
    so it will default to "Closed". Likewise, if there is no tag,
    the field name will be used.
*   ID is marked as the primary key. Currently only integer primary
    keys are supported. This is only relevant to Load, Save, Insert,
    and Update, a few of the higher-level functions that need to
    understand primary keys.  sqlscan assumes that pk fields have an
    autoincrement mechanism set in the database.
*   Created is marked with "localtime". This means that it will be
    converted to UTC when being saved, and back to the local time
    zone when being loaded.
*   Closed has a column name of "Closed", since the tag did not
    specify anything different. Closed is marked as "localtimez".
    This has the same properties as "localtime", except that the
    zero time will be saved in the database as a null column (and
    null values will be loaded as the zero time value).

sqlscan provides a few high-level functions (note: Db is an
interface that works with a *sql.Db or a *sql.Tx):

*   Load(db Db, table string, pk int, dst interface{}) error

    This loads a single record by its primary key. For example:

        elt := new(Person)
        err = sqlscan.Load(db, "person", 15, elt)

    db can be a *sql.Db or a *sql.Tx. The table is the name of the
    table, pk is the primary key value, and dst is a pointer to the
    struct where it should be stored.

    Note: this call requires that the struct have an integer primary
    key field marked.

*   Insert(db Db, table string, src interface{}) error

    This inserts a new row into the database. If the struct value
    has a primary key field, it must be zero (and will be omitted
    from the insert statement, prompting a default autoincrement
    value).

    Note: this call requires that the struct have an integer primary
    key field marked.

*   Update(db Db, table string, src interface{}) error

    This updates an existing row. It must have a primary key, which
    must be non-zero.

    Note: this call requires that the struct have an integer primary
    key field marked.

*   Save(db Db, table string, src interface{}) error

    Pick Insert or Update automatically. If there is a non-zero
    primary key present, it uses Update, otherwise it uses Insert.

    Note: this call requires that the struct have an integer primary
    key field marked.

*   QueryRow(db Db, dst interface{}, query string, args ...interface) error

    Perform the given query, and scan the single-row result into
    dst, which must be a pointer to a struct.

    For example:

        elt := new(Person)
        err := sqlscan.QueryRow(db, elt, "select * from person where name = ?", "bob")

*   QueryAll(db Db, dst interface{}, query string, args ...interface) error

    Perform the given query, and scan the results into dst, which
    must be a pointer to a slice of structs.

    For example:

        var people []Person
        err := sqlscan.QueryAll(db, &people, "select * from person")

*   Scan(rows *sql.Rows, dst interface{}) error

    Scans a single row of data into a struct, complete with
    meddling. Can be called repeatedly to walk through all of the
    rows in a result set. Returns sql.ErrNoRows when there is no
    more data.

*   ScanRow(rows *sql.Rows, dst interface{}) error

    Similar to Scan, but guarantees that the rows object
    is closed when it returns. Also returns sql.ErrNoRows if there
    was no row.

*   ScanAll(rows *sql.Rows, dst interface{}) error

    Expects a pointer to a slice of structs, and appends as
    many elements as it finds in the row set. Closes the row set
    when it is finished. Does not return sql.ErrNoRows on an empty
    set; instead it just does not add anything to the slice.


Meddlers
--------

sqlscan has a feature called "meddlers". A meddler is a handler that
gets to meddle with a field before it is saved, or when it is
loaded. "localtime" and "localtimez" are examples of built-in
meddlers. The full list of built-in meddlers includes:

*   identity: the default meddler, which does not do anything

*   localtime: for time.Time and *time.Time fields. Converts the
    value to UTC on save, and back to the local time zone on loads.
    To set your local time zone, use something like:

        os.Setenv("TZ", "America/Denver")

    in your initial setup, before you start using time functions.

*   localtimez: same, but only for time.Time, and treats the zero
    time as a null field (converts both ways)

*   utctime: similar to localtime, but keeps the value in UTC on
    loads. This ensures that the time is always coverted to UTC on
    save, which is the sane way to save time values in a database.

*   utctimez: same, but with zero time means null.

*   zeroisnull: for other types where a zero value should be
    inserted as null, and null values should be read as zero values.
    Works for integer, unsigned integer, float, complex number, and
    string types. Note: not for pointer types.

*   json: marshals the field value into JSON when saving, and
    unmarshals on load.

*   jsongzip: same, but compresses using gzip on save, and
    uncompresses on load
    
You can implement custom meddlers as well by implementing the
Meddler interface. See the existing implementations in medder.go for
examples.


Lower-level functions
---------------------

If you are using more complex queries and just want to reduce the
tedium of reading and writing values, there are some lower-level
helper functions as well. See the package docs for details, and
see the implementations of the higher-level functions to see how
they are used.
