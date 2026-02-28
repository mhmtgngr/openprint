# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e5]:
    - img [ref=e7]
    - heading "OpenPrint Cloud" [level=1] [ref=e9]
    - paragraph [ref=e10]: Print from anywhere, to any printer
  - generic [ref=e11]:
    - heading "Sign in to your account" [level=2] [ref=e12]
    - generic [ref=e13]: invalid credentials
    - generic [ref=e14]:
      - generic [ref=e15]:
        - generic [ref=e16]: Email Address
        - textbox "Email Address" [ref=e17]:
          - /placeholder: you@example.com
          - text: admin@example.com
      - generic [ref=e18]:
        - generic [ref=e19]: Password
        - textbox "Password" [ref=e20]:
          - /placeholder: ••••••••
          - text: AdminPassword123!
      - button "Sign In" [ref=e21] [cursor=pointer]
    - button "Don't have an account? Sign up" [ref=e23] [cursor=pointer]
  - paragraph [ref=e24]: By continuing, you agree to our Terms of Service and Privacy Policy.
```