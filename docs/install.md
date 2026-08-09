# Installing and updating

Every release ships two forms of the same application. Pick either — the
in-app updater works the same way for both.

| | Windows | macOS | Linux |
| --- | --- | --- | --- |
| Installer | `StrikeMan-windows-installer.exe` | `StrikeMan-macos.dmg` | – |
| Portable | `StrikeMan-windows.zip` | `StrikeMan-macos.zip` | `StrikeMan-linux.tar.gz` |

## Windows

The installer needs **no administrator rights**. It installs into
`%LOCALAPPDATA%\Programs\StrikeMan`, creates a Start menu and desktop
shortcut, and registers an uninstall entry under your own user account, so it
shows up in "Apps & features" as usual. Because it lives in your profile, the
in-app updater can replace it later without a UAC prompt.

The portable zip is just the executable — unpack it anywhere and run it. Your
servers and settings live in `%APPDATA%\StrikeMan\config.json` either way
(passwords go to Windows Credential Manager), so you can switch between the
two without losing anything.

### "Windows protected your PC"

That blue SmartScreen dialog appears because the download is not signed by a
certificate Microsoft recognises. It is expected. Click **More info → Run
anyway**.

StrikeMan is not code signed, and this is deliberate: a certificate costs a few
hundred euros a year, must live on a hardware token or cloud HSM, and — for an
ordinary Organisation Validation certificate — *would not even remove the
warning* until the publisher has built up download reputation. Microsoft's
cheap alternative, Azure Artifact Signing, is currently limited to
organisations and to individuals in the USA and Canada, so an EU individual
cannot use it at all.

If you would rather verify the download than trust it, every asset has a
`.sha256` file beside it:

```powershell
Get-FileHash .\StrikeMan-windows.zip -Algorithm SHA256
# compare with the contents of StrikeMan-windows.zip.sha256
```

## macOS

Open the `.dmg` and drag StrikeMan to Applications, or unpack the zip.

The app is not notarised (that needs an Apple Developer membership at $99 a
year), so the first launch is blocked. Right-click the app and choose **Open**,
then confirm — macOS remembers the decision. If Gatekeeper insists the app is
damaged, clear the quarantine flag:

```sh
xattr -dr com.apple.quarantine /Applications/StrikeMan.app
```

## Linux

```sh
tar -xzf StrikeMan-linux.tar.gz
./StrikeMan
```

Nothing to sign or approve. The binary needs a WebKit runtime
(`libwebkit2gtk-4.1`), which most desktop distributions already have.

## Verifying a download

```sh
sha256sum -c StrikeMan-linux.tar.gz.sha256   # Linux
shasum -a 256 StrikeMan-macos.zip            # macOS, compare by eye
```

## Updating

The Server card shows StrikeMan's own version and checks GitHub for a newer
release when the card refreshes or when you press **Check**. If one exists, an
**Update** button appears: StrikeMan downloads the release archive, verifies it
against the published `.sha256`, replaces its own executable and restarts. The
CS2 server is not touched.

A locally built copy reports its version as `local build` and is never offered
an update, so development builds do not overwrite themselves.

If an update ever goes wrong, the previous executable is kept next to the new
one as `StrikeMan.exe.old` until the next start — renaming it back gives you
the old version.
