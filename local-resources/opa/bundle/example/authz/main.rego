package example.authz

default allowed = false

allowed {
    input.user.groups[_] == "group1"
}
