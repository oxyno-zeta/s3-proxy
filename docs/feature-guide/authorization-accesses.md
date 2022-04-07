# Authorization accesses: How it works ?

The authorization accesses list coming from [ResourceHeaderOIDC](../configuration//structure.md#resourceheaderoidc) are accesses matrix by group or email. If not set, authenticated users will be authorized (no group or email validation will be performed if `authorizationOPAServer` isn't set).

Moreover, this is based on the "OR" principle. Another way to say it is: you are authorized as soon as 1 thing (email or group) is matching.

The example below explain this in detail.

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

- Jean Dupont with `group1` and `group2`
- Astérix with `group1` and `group3`
- Obélix with `group3`

Accesses will be:

- Jean Dupont: Ok because he is in `group1` (and `group2` but this one isn't matching the first)
- Astérix: Ok because he is in `group1`
- Obélix: Forbidden because he isn't in any of `group1` or `group2`

To conclude, if you want to have a **AND** accesses list (following the example before, only Jean Dupont is authorized), you will have to change the authorization mechanism to [OPAServerAuthorization](../configuration/structure.md#opaserverauthorization) and check feature guide [here](./opa.md).
