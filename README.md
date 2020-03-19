# An Encrypted Journaling CLI

```
go get github.com/jbpratt78/jrnl
```

This is currently only functioning for Vim, long term I would like to add support for additional editors. It uses AES encryption and requires a 32 byte key. Entries are dated and stored by default in `$HOME/.config`.
