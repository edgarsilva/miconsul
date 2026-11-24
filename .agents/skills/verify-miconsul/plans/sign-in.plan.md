# Sign in, sign up and password reset

Covers: internal/services/auth/, internal/views/authloginpage.templ, internal/views/authsignuppage.templ, internal/views/authresetpasswordpage.templ, internal/views/authchangepassword.templ, internal/views/landingpage.templ, internal/views/cmpnavbar.templ
Starts: a fresh stack, not signed in. Source `drive.sh` first. This plan creates `verify.user@seed.local`, which the admin plan uses.

## 1. Wrong password
Do: open /signin, fill the first text box with `admin@seed.local` and the second with `Wrong-pass-1`, click "Sign in"
Expect: still on /signin; an alert reads "incorrect email and password combination"

## 2. Sign in as the seed admin
Do: `signin`
Expect: the path is /dashboard?timeframe=day; the navbar has links "Admin" and "Sign out"

## 3. Sign out, then a protected page
Do: click "Sign out", then open /dashboard
Expect: both end on /signin

## 4. Sign up
Do: open /signup, fill the three text boxes (email `verify.user@seed.local`, password `Verify-Pass-123`, the same again), click "Sign up"
Expect: the page stays on /signup with the alert "failed to send confirmation email, please try again" (email is off in the stack); the database has the user with a `confirm_email_token`: `db "select confirm_email_token from users where email='verify.user@seed.local'"`

## 5. Confirm the email
Do: open /signup/confirm/<token> with that token
Expect: the path is /signin with the message "Email confirmed, you should be able to signin now"; the token is cleared in the database

## 6. Sign in as a regular user
Do: sign in as `verify.user@seed.local` / `Verify-Pass-123`
Expect: /dashboard?timeframe=day; the navbar has "Sign out" and no "Admin" link; opening /admin ends on /signin; `curl` with that user's session (see step 9) gets 403 from /api/admin/runtime

## 7. Reset the password
Do: sign out, open /signin, click "Forgot password?", fill the email `verify.user@seed.local`, click "Reset my password"
Expect: the path stays /resetpassword with "check your email for a reset password link"; the user has a `reset_token`; then open /resetpassword/change/<token>, fill both text boxes with `New-Pass-456`, click "Change password": the path is /signin

## 8. Sign in with the new password
Do: sign in as `verify.user@seed.local` / `New-Pass-456`
Expect: /dashboard?timeframe=day

## 9. API sign-in
Do: `curl -c jar.txt -H "Origin: $BASE" -H "Content-Type: application/json" -d '{"email":"admin@seed.local","password":"Admin123!"}' $BASE/api/auth/signin`, then `curl -b jar.txt $BASE/api/admin/runtime`
Expect: `{"ok":true}` and a `session_id` cookie in `jar.txt`; the second call returns the runtime JSON. This route exists only while Logto is off.
