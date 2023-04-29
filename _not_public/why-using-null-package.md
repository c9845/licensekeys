# Intro:
Why am I using `null` package instead of `sql.Null...` 


# Tags (for searching repo):
"gopkg.in/guregu/null.v3"
null.Int


# Probem & Error Encountered:
When trying to marshal a stringified JSON blob from an HTTP request into a struct with `sql.Null...` fields, an error occurs if a field in the struct is missing from the JSON blob.

Given a JSON blob such as:

```
{
    TeamID: 0
}
```

An error message such as "cannot unmarshal number into Go struct field User.TeamID of type sql.NullInt64 Could not parse data to add user" occurs.

With the null package, you can unmarshal successfully whether the field is present in the JSON blob, if the field is missing, and if the field has a valid value. I don't think the sql package was designed to allow use in this manner (unmarshalling JSON into a struct).
