description
-----------

A tool to tag mail. Configuration is loaded from a JSON file.

notmuch-filter will load a collection of filter from notmuch-filter.json and
apply tag depending on the result.

It will also:

* drop `tag:inbox` tag from thread with `tag:killed` message.
* remove `tag:inbox` from messages with `tag:seen`
* add `tag:inbox` to complet the thread if a new message show up

installation
------------

You will need Go and notmuch Go binding.

notmuch-filter.json has to be copied in your Maildir.

The Maildir default location is `$HOME/Maildir` or `$NOTMUCH_CONFIG`.

configuration
-------------

* Field: the header field of the mail
* Pattern: an regular expression
* Tags: tag to add or to remove

```json
    {"Field": "X-BeenThere", "Pattern": "obnam@flarn.net", "Tags": "+list +obnam -inbox" }
```

usage
-----

Edit `$HOME/.notmuch-config` and ensure new files will get the `new` tag:

    [new]
    tags=new

    $ getmail/fetchmail/offlineimap/whatever
    $ notmuch new
    $ notmuch-filter

license
-------

BSD, See LICENSE file.

contact
-------

* Gonéri Le Bouder <goneri@lebouder.net>
