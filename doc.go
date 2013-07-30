/*
<h1>sqlscan</h1>

<p>A small toolkit to automatically scan sql results into structs and
generate insert and update statements based on structs.</p>

<p>Package docs: <a href="http://godoc.org/github.com/russross/sqlscan">http://godoc.org/github.com/russross/sqlscan</a></p>

<p>This is currently designed for Sqlite, MySQL, and PostgreSQL, though
it has not been tested on PostgreSQL. If you test it, please let me
know if it works or not and I will update this README.</p>

<p>To use with PostgreSQL, set the following:</p>

<pre><code>sqlscan.Quote = &quot;\&quot;&quot;
sqlscan.Placeholder = &quot;$1&quot;
sqlscan.PostgreSQL = true
</code></pre>

<h2>High-level functions</h2>

<p>sqlscan does not create or alter tables. It just provides a little
glue to make it easier to read and write structs as SQL rows. Start
by annotating a struct:</p>

<pre><code class="go">type Person struct {
    ID      int       `sqlscan:&quot;id,pk&quot;`
    Name    string    `sqlscan:&quot;name&quot;`
    Created time.Time `sqlscan:&quot;created,localtime&quot;`
    Closed  time.Time `sqlscan:&quot;,localtimez&quot;`
}
</code></pre>

<p>Notes about this example:</p>

<ul>
<li>Non-public fields are ignored by sqlscan</li>
<li>If the optional tag is provided, the first field is the database
column name. Note that &ldquo;Closed&rdquo; does not provide a column name,
so it will default to &ldquo;Closed&rdquo;. Likewise, if there is no tag,
the field name will be used.</li>
<li>ID is marked as the primary key. Currently only integer primary
keys are supported. This is only relevant to Load, Save, Insert,
and Update, a few of the higher-level functions that need to
understand primary keys.  sqlscan assumes that pk fields have an
autoincrement mechanism set in the database.</li>
<li>Created is marked with &ldquo;localtime&rdquo;. This means that it will be
converted to UTC when being saved, and back to the local time
zone when being loaded.</li>
<li>Closed has a column name of &ldquo;Closed&rdquo;, since the tag did not
specify anything different. Closed is marked as &ldquo;localtimez&rdquo;.
This has the same properties as &ldquo;localtime&rdquo;, except that the
zero time will be saved in the database as a null column (and
null values will be loaded as the zero time value).</li>
</ul>

<p>sqlscan provides a few high-level functions (note: DB is an
interface that works with a *sql.DB or a *sql.Tx):</p>

<ul>
<li><p>Load(db DB, table string, pk int, dst interface{}) error</p>

<p>This loads a single record by its primary key. For example:</p>

<pre><code>elt := new(Person)
err = sqlscan.Load(db, &quot;person&quot;, 15, elt)
</code></pre>

<p>db can be a *sql.DB or a *sql.Tx. The table is the name of the
table, pk is the primary key value, and dst is a pointer to the
struct where it should be stored.</p>

<p>Note: this call requires that the struct have an integer primary
key field marked.</p></li>

<li><p>Insert(db DB, table string, src interface{}) error</p>

<p>This inserts a new row into the database. If the struct value
has a primary key field, it must be zero (and will be omitted
from the insert statement, prompting a default autoincrement
value).</p></li>

<li><p>Update(db DB, table string, src interface{}) error</p>

<p>This updates an existing row. It must have a primary key, which
must be non-zero.</p>

<p>Note: this call requires that the struct have an integer primary
key field marked.</p></li>

<li><p>Save(db DB, table string, src interface{}) error</p>

<p>Pick Insert or Update automatically. If there is a non-zero
primary key present, it uses Update, otherwise it uses Insert.</p>

<p>Note: this call requires that the struct have an integer primary
key field marked.</p></li>

<li><p>QueryRow(db DB, dst interface{}, query string, args &hellip;interface) error</p>

<p>Perform the given query, and scan the single-row result into
dst, which must be a pointer to a struct.</p>

<p>For example:</p>

<pre><code>elt := new(Person)
err := sqlscan.QueryRow(db, elt, &quot;select * from person where name = ?&quot;, &quot;bob&quot;)
</code></pre></li>

<li><p>QueryAll(db DB, dst interface{}, query string, args &hellip;interface) error</p>

<p>Perform the given query, and scan the results into dst, which
must be a pointer to a slice of structs.</p>

<p>For example:</p>

<pre><code>var people []Person
err := sqlscan.QueryAll(db, &amp;people, &quot;select * from person&quot;)
</code></pre></li>

<li><p>Scan(rows *sql.Rows, dst interface{}) error</p>

<p>Scans a single row of data into a struct, complete with
meddling. Can be called repeatedly to walk through all of the
rows in a result set. Returns sql.ErrNoRows when there is no
more data.</p></li>

<li><p>ScanRow(rows *sql.Rows, dst interface{}) error</p>

<p>Similar to Scan, but guarantees that the rows object
is closed when it returns. Also returns sql.ErrNoRows if there
was no row.</p></li>

<li><p>ScanAll(rows *sql.Rows, dst interface{}) error</p>

<p>Expects a pointer to a slice of structs, and appends as
many elements as it finds in the row set. Closes the row set
when it is finished. Does not return sql.ErrNoRows on an empty
set; instead it just does not add anything to the slice.</p></li>
</ul>

<h2>Meddlers</h2>

<p>sqlscan has a feature called &ldquo;meddlers&rdquo;. A meddler is a handler that
gets to meddle with a field before it is saved, or when it is
loaded. &ldquo;localtime&rdquo; and &ldquo;localtimez&rdquo; are examples of built-in
meddlers. The full list of built-in meddlers includes:</p>

<ul>
<li><p>identity: the default meddler, which does not do anything</p></li>

<li><p>localtime: for time.Time and *time.Time fields. Converts the
value to UTC on save, and back to the local time zone on loads.
To set your local time zone, use something like:</p>

<pre><code>os.Setenv(&quot;TZ&quot;, &quot;America/Denver&quot;)
</code></pre>

<p>in your initial setup, before you start using time functions.</p></li>

<li><p>localtimez: same, but only for time.Time, and treats the zero
time as a null field (converts both ways)</p></li>

<li><p>utctime: similar to localtime, but keeps the value in UTC on
loads. This ensures that the time is always coverted to UTC on
save, which is the sane way to save time values in a database.</p></li>

<li><p>utctimez: same, but with zero time means null.</p></li>

<li><p>zeroisnull: for other types where a zero value should be
inserted as null, and null values should be read as zero values.
Works for integer, unsigned integer, float, complex number, and
string types. Note: not for pointer types.</p></li>

<li><p>json: marshals the field value into JSON when saving, and
unmarshals on load.</p></li>

<li><p>jsongzip: same, but compresses using gzip on save, and
uncompresses on load</p></li>
</ul>

<p>You can implement custom meddlers as well by implementing the
Meddler interface. See the existing implementations in medder.go for
examples.</p>

<h2>Lower-level functions</h2>

<p>If you are using more complex queries and just want to reduce the
tedium of reading and writing values, there are some lower-level
helper functions as well. See the package docs for details, and
see the implementations of the higher-level functions to see how
they are used.</p>
*/
package sqlscan
