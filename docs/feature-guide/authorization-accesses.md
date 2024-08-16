# Authorization accesses: How it works ?

## General

The authorization accesses list coming from [ResourceHeaderOIDC](../configuration//structure.md#resourceheaderoidc) are accesses matrix by group or email. If not set, authenticated users will be authorized (no group or email validation will be performed if `authorizationOPAServer` isn't set).

Moreover, this is based on the "OR" principle. Another way to say it is: you are authorized as soon as 1 thing (email or group) is matching.

To conclude, if you want to have a **AND** accesses list (following the example before, only Jean Dupont is authorized), you will have to change the authorization mechanism to [OPAServerAuthorization](../configuration/structure.md#opaserverauthorization) and check feature guide [here](./opa.md).

## Examples

### Empty list

Example of authorization accesses configuration:

```yaml
targets:
  target1:
    resources:
      - path: /*
        provider: provider1
        oidc:
          authorizationAccesses: []
     bucket:
       ...
```

We consider those users:

- Jean Dupont with `group1` and `group2` groups
- Astérix with `group1` and `group3` groups
- Obélix with `group3` group

Accesses will be:

- Jean Dupont: Authorized because list is empty
- Astérix: Authorized because list is empty
- Obélix: Authorized because list is empty

### Group matching

Example of authorization accesses configuration:

```yaml
targets:
  target1:
    resources:
      - path: /*
        provider: provider1
        oidc:
          authorizationAccesses:
            - group: group1
            - group: group2
     bucket:
       ...
```

We consider those users:

- Jean Dupont with `group1` and `group2` groups
- Astérix with `group1` and `group3` groups
- Obélix with `group3` group

Accesses will be:

- Jean Dupont: Ok because he is in `group1` (and `group2` but this one isn't matching the first)
- Astérix: Ok because he is in `group1`
- Obélix: Forbidden because he isn't in any of `group1` or `group2`

### Group regex matching

Example of authorization accesses configuration:

```yaml
targets:
  target1:
    resources:
      - path: /*
        provider: provider1
        oidc:
          authorizationAccesses:
            - group: valid.*
              regex: true
     bucket:
       ...
```

We consider those users:

- Jean Dupont with `valid1` and `valid2` groups
- Astérix with `valid1` and `group3` groups
- Obélix with `group3` group

Accesses will be:

- Jean Dupont: Ok because he is in `valid1` and `valid2`
- Astérix: Ok because he is in `valid1`
- Obélix: Forbidden because he isn't in any of `valid1` or `valid2`

### Email matching

Example of authorization accesses configuration:

```yaml
targets:
  target1:
    resources:
      - path: /*
        provider: provider1
        oidc:
          authorizationAccesses:
            - email: jean.dupont@fake.com
     bucket:
       ...
```

We consider those users:

- Jean Dupont with `jean.dupont@fake.com` email
- Astérix with `asterix@fake.com` email
- Obélix with `obelix@fake.com` email

Accesses will be:

- Jean Dupont: authorized
- Astérix: forbidden
- Obélix: forbidden

### Email regex matching

Example of authorization accesses configuration:

```yaml
targets:
  target1:
    resources:
      - path: /*
        provider: provider1
        oidc:
          authorizationAccesses:
            - email: .*@fake.com
              regex: true
     bucket:
       ...
```

We consider those users:

- Jean Dupont with `jean.dupont@fake.com` email
- Astérix with `asterix@fake.com` email
- Obélix with `obelix@another.com` email

Accesses will be:

- Jean Dupont: authorized
- Astérix: authorized
- Obélix: forbidden

### Forbidden case

<!-- prettier-ignore-start -->
!!! note

    This have been done because there isn't any way of doing a negative match on regex. The only way is to match a regex with a forbidden flag enabled.
<!-- prettier-ignore-end -->

Example of authorization accesses configuration:

```yaml
targets:
  target1:
    resources:
      - path: /*
        provider: provider1
        oidc:
          authorizationAccesses:
            - email: asterix@fake.com
              regex: true
              forbidden: true
            - email: .*@fake.com
              regex: true
     bucket:
       ...
```

We consider those users:

- Jean Dupont with `jean.dupont@fake.com` email
- Astérix with `asterix@fake.com` email
- Obélix with `obelix@another.com` email

Accesses will be:

- Jean Dupont: authorized
- Astérix: forbidden because it is marked as forbidden and it is the first in the list
- Obélix: forbidden
