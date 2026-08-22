// Command winres writes a Windows VERSIONINFO, manifest, and icon .syso
// next to the call-detect main package so go build embeds them.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/josephspurrier/goversioninfo"

	"github.com/scallister/call-detect/internal/tray"
	"github.com/scallister/call-detect/internal/version"
)

func main() {
	ver := flag.String("version", version.Version, "release string such as v0.1.0")
	out := flag.String("o", "cmd/call-detect/rsrc_windows_amd64.syso", "output .syso path")
	flag.Parse()
	if err := writeSyso(*ver, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeSyso(ver, out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "call-detect-winres-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	iconPath := filepath.Join(dir, "icon.ico")
	if err := os.WriteFile(iconPath, tray.IdleICO(), 0o644); err != nil {
		return err
	}
	maj, min, pat, bld := parseVersion(ver)
	ident := fmt.Sprintf("%d.%d.%d.%d", maj, min, pat, bld)
	manifestPath := filepath.Join(dir, "app.manifest")
	if err := os.WriteFile(manifestPath, []byte(windowsManifest(ident)), 0o644); err != nil {
		return err
	}

	vi := goversioninfo.VersionInfo{
		FixedFileInfo: goversioninfo.FixedFileInfo{
			FileVersion:    goversioninfo.FileVersion{Major: maj, Minor: min, Patch: pat, Build: bld},
			ProductVersion: goversioninfo.FileVersion{Major: maj, Minor: min, Patch: pat, Build: bld},
			FileFlagsMask:  "3f",
			FileFlags:      "00",
			FileOS:         "040004",
			FileType:       "01",
			FileSubType:    "00",
		},
		StringFileInfo: goversioninfo.StringFileInfo{
			Comments:         "https://github.com/scallister/call-detect",
			CompanyName:      "call-detect",
			FileDescription:  "Detects microphone and webcam use",
			FileVersion:      ident,
			InternalName:     "call-detect",
			LegalCopyright:   "MIT License",
			OriginalFilename: "call-detect.exe",
			ProductName:      "call-detect",
			ProductVersion:   ident,
		},
		VarFileInfo: goversioninfo.VarFileInfo{
			Translation: goversioninfo.Translation{LangID: goversioninfo.LngUSEnglish, CharsetID: goversioninfo.CsUnicode},
		},
		IconPath:     iconPath,
		ManifestPath: manifestPath,
	}
	vi.Build()
	vi.Walk()
	return vi.WriteSyso(out, "amd64")
}

func parseVersion(v string) (major, minor, patch, build int) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	if v == "" || v == "dev" {
		return 0, 0, 0, 0
	}
	parts := strings.Split(v, ".")
	nums := []*int{&major, &minor, &patch, &build}
	for i := 0; i < len(parts) && i < len(nums); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			break
		}
		*nums[i] = n
	}
	return major, minor, patch, build
}

func windowsManifest(fileVersion string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity version="` + fileVersion + `" processorArchitecture="*" name="call-detect" type="win32"/>
  <description>Detects microphone and webcam use</description>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
      <supportedOS Id="{1f676c76-80e1-4239-95bb-83d0f6d0da78}"/>
      <supportedOS Id="{4a2f28e3-53b9-4441-ba9c-d69d4a4a6e38}"/>
      <supportedOS Id="{35138b9a-5d96-4fbd-8e2d-a2440225f93a}"/>
      <supportedOS Id="{e2011457-1546-43c5-a5fe-008deee3d3f0}"/>
    </application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>
      <longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware>
    </windowsSettings>
  </application>
</assembly>
`
}
