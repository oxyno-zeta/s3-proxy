# AWS IAM Policy

This application will need some rights on AWS.

This is an example of a policy that allow access to `<bucket-name>` and files in it:

```js
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        // Needed for GET API/Action
        "s3:ListBucket",
        "s3:GetObject",
        // Needed for PUT API/Action
        "s3:PutObject",
        // Needed for DELETE API/Action
        "s3:DeleteObject"
      ],
      "Resource": ["arn:aws:s3:::<bucket-name>", "arn:aws:s3:::<bucket-name>/*"]
    }
  ]
}
```
