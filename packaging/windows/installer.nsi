; NSIS installer for «Костыль 1С» (Windows 10/11 x64).
Unicode true
!include "MUI2.nsh"
!include "x64.nsh"

!define APPNAME "Костыль 1С"
!define REGKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\Kostil1C"

Name "${APPNAME} ${VERSION}"
OutFile "${OUTFILE}"
InstallDir "$PROGRAMFILES64\Kostil1C"
InstallDirRegKey HKLM "Software\Kostil1C" "InstallDir"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
BrandingText "Горшков Сергей Владимирович — nookbat.ru"

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${APPNAME}"
VIAddVersionKey "CompanyName" "Горшков Сергей Владимирович"
VIAddVersionKey "LegalCopyright" "Горшков Сергей Владимирович, https://nookbat.ru/"
VIAddVersionKey "FileDescription" "Установщик ${APPNAME}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"

!define MUI_ICON "${SRCDIR}\kostil1c.ico"
!define MUI_UNICON "${SRCDIR}\kostil1c.ico"
!define MUI_FINISHPAGE_RUN "$INSTDIR\kostil1c.exe"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "Russian"

Function .onInit
  ${IfNot} ${RunningX64}
    MessageBox MB_ICONSTOP "Требуется 64-разрядная Windows 10 или 11."
    Abort
  ${EndIf}
  SetRegView 64
FunctionEnd

Section "Install"
  SetOutPath "$INSTDIR"
  File "${SRCDIR}\kostil1c.exe"
  File "${SRCDIR}\kostil1c.ico"
  WriteUninstaller "$INSTDIR\uninstall.exe"
  CreateDirectory "$SMPROGRAMS\${APPNAME}"
  CreateShortcut "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk" "$INSTDIR\kostil1c.exe"
  CreateShortcut "$SMPROGRAMS\${APPNAME}\Удалить ${APPNAME}.lnk" "$INSTDIR\uninstall.exe"
  CreateShortcut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\kostil1c.exe"
  WriteRegStr HKLM "Software\Kostil1C" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${REGKEY}" "DisplayName" "${APPNAME}"
  WriteRegStr HKLM "${REGKEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${REGKEY}" "Publisher" "Горшков Сергей Владимирович"
  WriteRegStr HKLM "${REGKEY}" "URLInfoAbout" "https://nookbat.ru/"
  WriteRegStr HKLM "${REGKEY}" "DisplayIcon" "$INSTDIR\kostil1c.ico"
  WriteRegStr HKLM "${REGKEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegDWORD HKLM "${REGKEY}" "NoModify" 1
  WriteRegDWORD HKLM "${REGKEY}" "NoRepair" 1
SectionEnd

Function un.onInit
  SetRegView 64
FunctionEnd

Section "Uninstall"
  Delete "$INSTDIR\kostil1c.exe"
  Delete "$INSTDIR\kostil1c.ico"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk"
  Delete "$SMPROGRAMS\${APPNAME}\Удалить ${APPNAME}.lnk"
  RMDir "$SMPROGRAMS\${APPNAME}"
  Delete "$DESKTOP\${APPNAME}.lnk"
  DeleteRegKey HKLM "${REGKEY}"
  DeleteRegKey HKLM "Software\Kostil1C"
SectionEnd
