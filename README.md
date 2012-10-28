description
-----------

A tool to tag mail. Configuration is loaded from a JSON file.

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

license
-------

BSD, See LICENSE file.

contact
-------

* Gonéri Le Bouder <goneri@rulezlan.org>
