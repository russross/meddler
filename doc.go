/*
Package github.com/russross/sqlscan is a small toolkit to automatically scan
sql results into structs and generate insert and update statements based on
structs. Package docs are available at:

	http://godoc.org/github.com/russross/sqlscan

This is currently designed for Sqlite, MySQL, and PostgreSQL, though
it has not been tested on PostgreSQL. If you test it, please let me
know if it works or not and I will update this note.

To use with PostgreSQL, set the following:

	sqlscan.Quote = "\""
	sqlscan.Placeholder = "$1"
	sqlscan.PostgreSQL = true


High-level functions
--------------------

sqlscan does not create or alter tables. It just provides a little
glue to make it easier to read and write structs as SQL rows. Start
by annotating a struct:

	type Person struct {
		ID      int       `sqlscan:"id,pk"`
		Name    string    `sqlscan:"name"`
		Created time.Time `sqlscan:"created,localtime"`
		Closed  time.Time `sqlscan:",localtimez"`
	}

Notes about this example:

* Non-public fields are ignored by sqlscan

* If the optional tag is provided, the first field is the database
column name. Note that "Closed" does not provide a column name,
so it will default to "Closed". Likewise, if there is no tag,
the field name will be used.

* ID is marked as the primary key. Currently only integer primary
keys are supported. This is only relevant to Load, Save, Insert,
and Update, a few of the higher-level functions that need to
understand primary keys.  sqlscan assumes that pk fields have an
autoincrement mechanism set in the database.

* Created is marked with "localtime". This means that it will be
converted to UTC when being saved, and back to the local time
zone when being loaded.

* Closed has a column name of "Closed", since the tag did not
specify anything different. Closed is marked as "localtimez".
This has the same properties as "localtime", except that the
zero time will be saved in the database as a null column (and
null values will be loaded as the zero time value).

sqlscan provides a few high-level functions (note: DB is an
interface that works with a *sql.DB or a *sql.Tx):

	Load(db DB, table string, pk int, dst interface{}) error
	Insert(db DB, table string, src interface{}) error
	Update(db DB, table string, src interface{}) error
	Save(db DB, table string, src interface{}) error
	QueryRow(db DB, dst interface{}, query string, args ...interface) error
	QueryAll(db DB, dst interface{}, query string, args ...interface) error
	Scan(rows *sql.Rows, dst interface{}) error
	ScanRow(rows *sql.Rows, dst interface{}) error
	ScanAll(rows *sql.Rows, dst interface{}) error


Meddlers
--------

sqlscan has a feature called "meddlers". A meddler is a handler that
gets to meddle with a field before it is saved, or when it is
loaded. "localtime" and "localtimez" are examples of built-in
meddlers. The full list of built-in meddlers includes:

* identity: the default meddler, which does not do anything

* localtime: for time.Time and *time.Time fields. Converts the
value to UTC on save, and back to the local time zone on loads.
To set your local time zone, use something like:

	os.Setenv("TZ", "America/Denver")

in your initial setup, before you start using time functions.

* localtimez: same, but only for time.Time, and treats the zero
time as a null field (converts both ways)

* utctime: similar to localtime, but keeps the value in UTC on
loads. This ensures that the time is always coverted to UTC on
save, which is the sane way to save time values in a database.

* utctimez: same, but with zero time means null.

* zeroisnull: for other types where a zero value should be
inserted as null, and null values should be read as zero values.
Works for integer, unsigned integer, float, complex number, and
string types. Note: not for pointer types.

* json: marshals the field value into JSON when saving, and
unmarshals on load.

* jsongzip: same, but compresses using gzip on save, and
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

*/
package sqlscan
