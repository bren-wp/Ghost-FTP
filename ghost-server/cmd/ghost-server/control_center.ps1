# Native Ghost FTP Server Control Center; no HTML, WebView or external network.
$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()
function Color([string]$hex){[System.Drawing.ColorTranslator]::FromHtml($hex)}
$bg=Color '#070E1A'; $navy=Color '#0B1E36'; $panel=Color '#0D192B'
$cyan=Color '#7EDEFF'; $white=Color '#EAF6FF'; $muted=Color '#A2B6CA'
$raised=Color '#111F34'; $accent=Color '#38ABFF'
$service='GhostFTPServer'
$exe=$env:GHOSTFTP_SERVER_EXE
if([string]::IsNullOrWhiteSpace($exe) -or !(Test-Path -LiteralPath $exe -PathType Leaf)){throw 'Ghost FTP Server executable not found'}
$work=[System.Windows.Forms.Screen]::PrimaryScreen.WorkingArea
$width=[Math]::Min(1080,[Math]::Max(650,$work.Width-80))
$height=[Math]::Min(700,[Math]::Max(450,$work.Height-80))
$form=New-Object System.Windows.Forms.Form
$form.Text='Ghost FTP Server — Control Center'
$form.Size=New-Object System.Drawing.Size($width,$height)
$form.MinimumSize=New-Object System.Drawing.Size(650,450)
$form.StartPosition='CenterScreen'
$form.FormBorderStyle='Sizable'
$form.MaximizeBox=$true
$form.AutoScaleMode='Dpi'
$form.Font=New-Object System.Drawing.Font('Segoe UI',10)
$form.ForeColor=$white
$form.BackColor=$bg

function Txt([string]$text,[int]$size,[bool]$bold){
 $label=New-Object System.Windows.Forms.Label
 $label.Text=$text
 $label.ForeColor=$white
 $label.AutoSize=$false
 $style=if($bold){[System.Drawing.FontStyle]::Bold}else{[System.Drawing.FontStyle]::Regular}
 $label.Font=New-Object System.Drawing.Font('Segoe UI',$size,$style)
 return $label
}
$side=New-Object System.Windows.Forms.Panel
$side.Dock='Left'
$side.Width=214
$side.BackColor=$navy
$form.Controls.Add($side)
$brand=Txt 'GHOST FTP' 17 $true
$brand.Location=New-Object System.Drawing.Point(16,17)
$brand.Size=New-Object System.Drawing.Size(190,30)
$side.Controls.Add($brand)
$brandCaption=Txt 'SERVER · NATIVE' 8 $true
$brandCaption.Location=New-Object System.Drawing.Point(17,48)
$brandCaption.ForeColor=$cyan
$brandCaption.Size=New-Object System.Drawing.Size(190,25)
$side.Controls.Add($brandCaption)
$foot=Txt 'LOCAL CONTROL · WINDOWS' 8 $false
$foot.ForeColor=$muted
$foot.Dock='Bottom'
$foot.Height=37
$side.Controls.Add($foot)

$workArea=New-Object System.Windows.Forms.Panel
$workArea.Dock='Fill'
$workArea.BackColor=$bg
$form.Controls.Add($workArea)
$side.BringToFront()
$header=New-Object System.Windows.Forms.Panel
$header.Dock='Top'
$header.Height=52
$header.BackColor=$panel
$workArea.Controls.Add($header)
$crumb=Txt 'Ghost FTP / Server / Overview' 10 $false
$crumb.Location=New-Object System.Drawing.Point(20,16)
$crumb.Size=New-Object System.Drawing.Size(530,25)
$crumb.ForeColor=$muted
$header.Controls.Add($crumb)
$page=New-Object System.Windows.Forms.Panel
$page.Dock='Fill'
$page.Padding=New-Object System.Windows.Forms.Padding(22)
$page.BackColor=$bg
$workArea.Controls.Add($page)
$header.BringToFront()
$heading=Txt 'Overview' 23 $true
$heading.Dock='Top'
$heading.Height=49
$page.Controls.Add($heading)
$subtitle=Txt 'Local service state; no fictional metrics or exposed secrets.' 10 $false
$subtitle.ForeColor=$muted
$subtitle.Dock='Top'
$subtitle.Height=38
$page.Controls.Add($subtitle)
$subtitle.BringToFront();$heading.BringToFront()

$toolbar=New-Object System.Windows.Forms.FlowLayoutPanel
$toolbar.Dock='Bottom';$toolbar.Height=55;$toolbar.WrapContents=$false
$toolbar.FlowDirection='LeftToRight';$toolbar.BackColor=$bg
$page.Controls.Add($toolbar)
function Button([string]$text){
 $b=New-Object System.Windows.Forms.Button
 $b.Text=$text
 $b.Size=New-Object System.Drawing.Size(118,37)
 $b.FlatStyle='Flat'
 $b.FlatAppearance.BorderColor=$accent
 $b.FlatAppearance.BorderSize=1
 $b.BackColor=$raised
 $b.ForeColor=$white
 $b.Margin=New-Object System.Windows.Forms.Padding(0,4,9,4)
 return $b
}
$start=Button 'Start service';$stop=Button 'Stop service';$refresh=Button 'Refresh'
$toolbar.Controls.AddRange([System.Windows.Forms.Control[]]@($start,$stop,$refresh))
$details=New-Object System.Windows.Forms.RichTextBox
$details.ReadOnly=$true
$details.Dock='Fill'
$details.BorderStyle='None'
$details.BackColor=$panel
$details.ForeColor=$white
$details.Font=New-Object System.Drawing.Font('Consolas',10)
$details.DetectUrls=$false
$page.Controls.Add($details)
$heading.BringToFront();$subtitle.BringToFront();$toolbar.BringToFront()

$script:current='Overview';$script:serviceStatus='Not installed'
function GetActualStatus {
 $sv=Get-Service -Name $service -ErrorAction SilentlyContinue
 if($null -eq $sv){return 'Not installed'}
 return $sv.Status.ToString()
}
function UpdateView([string]$name){
 $script:current=$name
 $script:serviceStatus=GetActualStatus
 $heading.Text=$name
 $crumb.Text='Ghost FTP / Server / '+$name
 $compiled=try{(& $exe -version | Out-String).Trim()}catch{'Version unavailable'}
 $common='Executable: '+$compiled+[Environment]::NewLine+'Service: '+$service+[Environment]::NewLine+
   'SCM status: '+$script:serviceStatus+[Environment]::NewLine+[Environment]::NewLine
 $message=switch($name){
   'Overview' {'This native window manages the real local service.'+[Environment]::NewLine+
     'The server only starts after explicit administrator provisioning of TLS keys, password file and filesystem root.'}
   'Sessions' {'The Windows Service Control Manager does not expose per-client FTPS sessions.'+
     [Environment]::NewLine+'This screen intentionally does not display made-up connections.'}
   'Security' {'TLS certificate, private key and password file are operator-managed.'+
     [Environment]::NewLine+'This window does not store, display or transmit secrets.'}
   'Limits' {'Default daemon policy: loopback explicit FTPS and read-only storage.'+
     [Environment]::NewLine+'Runtime flags are controlled by the actual installed service, not by this window.'}
   'Diagnostics' {'Management implementation: native Windows Forms (no WebView).'+
     [Environment]::NewLine+'If the service is not installed, use the administrator-only Install-GhostFTPServer.ps1 script.'}
 }
 $details.Text=$common+$message
 $subtitle.Text='Actual system state; no fabricated telemetry.'
 $start.Enabled=($script:serviceStatus -eq 'Stopped')
 $stop.Enabled=($script:serviceStatus -eq 'Running')
}
$names=@('Overview','Sessions','Security','Limits','Diagnostics')
for($i=0;$i -lt $names.Length;$i++){
 $b=Button $names[$i]
 $b.TextAlign='MiddleLeft'
 $b.Size=New-Object System.Drawing.Size(194,43)
 $b.Location=New-Object System.Drawing.Point(10,(100+$i*47))
 $b.BackColor=$navy
 $b.FlatAppearance.BorderSize=0
 $b.Tag=$names[$i]
 $b.Add_Click({UpdateView ([string]$this.Tag)})
 $side.Controls.Add($b)
}
$refresh.Add_Click({UpdateView $script:current})
$start.Add_Click({
 try{
  Start-Service -Name $service -ErrorAction Stop
  (Get-Service -Name $service).WaitForStatus('Running',[TimeSpan]::FromSeconds(20))
 }catch{
  [System.Windows.Forms.MessageBox]::Show('Service start failed. Check TLS paths, service permissions and Windows Event Viewer.'+
  [Environment]::NewLine+$_.Exception.Message,'Ghost FTP Server','OK','Error')|Out-Null
 }
 UpdateView $script:current
})
$stop.Add_Click({
 try{
  Stop-Service -Name $service -ErrorAction Stop
  (Get-Service -Name $service).WaitForStatus('Stopped',[TimeSpan]::FromSeconds(30))
 }catch{
  [System.Windows.Forms.MessageBox]::Show('Service stop failed.'+
  [Environment]::NewLine+$_.Exception.Message,'Ghost FTP Server','OK','Error')|Out-Null
 }
 UpdateView $script:current
})
$timer=New-Object System.Windows.Forms.Timer
$timer.Interval=4000
$timer.Add_Tick({if((GetActualStatus) -ne $script:serviceStatus){UpdateView $script:current}})
$form.Add_Shown({UpdateView 'Overview';$timer.Start()})
$form.Add_FormClosed({$timer.Stop();$timer.Dispose()})
[System.Windows.Forms.Application]::Run($form)
