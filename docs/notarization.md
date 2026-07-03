# macOS code signing & notarization — setup runbook

> **Status: dormant by decision (2026-07-03).** The owner has chosen **not** to
> activate notarization for now — it requires a paid Apple Developer account
> and the maintenance of signing credentials, and the `xattr` one-liner (and
> the install script's automatic quarantine strip) is an acceptable stopgap at
> this stage. The pipeline wiring below is left in place and inert; activating
> it later is purely additive (set the five secrets, tag a release) with no
> code change. This runbook is the how-to for that future day.

The release pipeline signs and notarizes the macOS binaries the way any
distributed Mac CLI does, so Gatekeeper stops flagging them and the
`xattr -d com.apple.quarantine` workaround becomes unnecessary. It uses
GoReleaser's built-in [quill](https://github.com/anchore/quill) — **no macOS
runner required**, it runs on the existing Ubuntu release job.

The `.goreleaser.yaml` and `release.yml` wiring is already in place and
**dormant**: releases work unchanged until the five secrets below exist, then
the next tag is signed automatically.

## One-time prerequisites (needs an Apple Developer account — $99/yr)

### 1. Developer ID Application certificate → `.p12`

1. Apple Developer → Certificates → **＋** → **Developer ID Application**.
2. Create a CSR (Keychain Access → Certificate Assistant → Request a
   Certificate from a CA → "Saved to disk"), upload it, download the `.cer`.
3. Double-click to import into Keychain, then **export** the cert *and its
   private key* together as a `.p12` with a password.

### 2. App Store Connect API key → `.p8`

App Store Connect → Users and Access → **Integrations / Keys** → **＋**.
- Role: **Developer** is sufficient for notarization.
- Download the `AuthKey_XXXXXXXX.p8` **once** (Apple won't let you re-download).
- Note the **Key ID** (in the filename) and the **Issuer ID** (UUID shown
  above the keys table).

## Encode and store the five secrets

Base64-encode the two files (the `-w0` avoids line wraps; on macOS `base64`
has no `-w`, so use the `| tr -d '\n'` form):

```sh
base64 < DeveloperID.p12       | tr -d '\n' | pbcopy   # -> MACOS_SIGN_P12
base64 < AuthKey_XXXXXXXX.p8   | tr -d '\n' | pbcopy   # -> MACOS_NOTARY_KEY
```

Add these as **repository secrets** (Settings → Secrets and variables →
Actions), or with `gh`:

```sh
gh secret set MACOS_SIGN_P12          # paste the .p12 base64
gh secret set MACOS_SIGN_PASSWORD     # the .p12 export password
gh secret set MACOS_NOTARY_KEY        # paste the .p8 base64
gh secret set MACOS_NOTARY_KEY_ID     # e.g. XXXXXXXXXX
gh secret set MACOS_NOTARY_ISSUER_ID  # the issuer UUID
```

## Activate

Nothing else changes. The next `git tag vX.Y.Z && git push origin vX.Y.Z`
triggers a release that signs both darwin binaries and submits them to Apple's
notary service (`wait: true`, up to 20m). The `enabled` guard
(`{{ isEnvSet "MACOS_SIGN_P12" }}`) turns the step on automatically once the
secret is present.

## Verify a notarized release

```sh
brew update && brew upgrade --cask YellowFoxH4XOR/tap/dwarpal
codesign -dv --verbose=4 "$(readlink -f "$(command -v dwarpal)")"   # Authority: Developer ID Application
spctl -a -vv -t install "$(readlink -f "$(command -v dwarpal)")"     # accepted; source=Notarized Developer ID
```

Once this is verified, drop the "not yet notarized / xattr" note from the
README's install section.

## Rotation & cost notes

- The `.p12` certificate expires (~5 years); the `.p8` API key does not but can
  be revoked. Rotating either = re-encode, re-set the secret. No code change.
- Everything runs on the Ubuntu runner — no paid macOS CI minutes. The only
  cost is the Apple Developer membership.
