; paste in browser: data:image/png;base64,{base64imgdata}
#Requires AutoHotkey v2.0
#SingleInstance Force

; Constants {
;log in to https://aistudio.google.com/ create new project, then create an API key.
global API_KEY := EnvGet("API_KEY") ;"USE YER OWN" 
;Folder to find images made
global OutputDir := A_ScriptDir "\img"
;defaut prompt
global proVal := "everyone stands on top of a large pile of burgers. the burgers deform under load."
;defaut negative prompt
global negVal := "distorted faces, blurry, distortion, low quality, watermarks, missing limbs, extra limbs, deformities, floating people or objects, cropping body parts"
; encourage sets up a 'character' or goal for the AI
global encourageEdt := "You are a professional image-restoration engine. Please apply the 'USER DIRECTIVE' while maintaining strict structural integrity. Focus on high-fidelity surface rendering and cinematic lighting. Ensure all facial features are sharp, clear and perfectly aligned with the reference without looking plastic. Resolve blur into crisp, clean, 8k-resolution details. Maintain 100% adherence to the subject's identity. If the directive involves clothing, ensure the new attire is rendered with realistic fabric textures and consistent coverage."
global encourageGen := "You are a world-class visual concept artist. Please transform the user's prompt into a vivid, high-fidelity masterpiece. Prioritize cinematic lighting, photorealistic textures, and perfect anatomical detail. Every output must be rendered with the clarity of an 8k digital sensor. Interpret abstract concepts as concrete, visually dense scenes. Ensure all subjects, especially faces and hands, are rendered with sharp focus and professional-grade definition."
; Models used in app, sometimes these will change
global MODEL1 := "gemini-2.5-flash-image" ; 1k
global MODEL2 := "gemini-3-pro-image-preview" ; 1k 2k 4k
global MODEL5 := "gemini-3.1-flash-image-preview" ; 1k 2k 4k
global MODEL3 := "imagen-4.0-generate-001" ; 2k
global MODEL4 := "imagen-4.0-ultra-generate-001" ; 2k
global hurl := "https://generativelanguage.googleapis.com/v1beta/models/"
global CheckInterval := 300000 ; 5 minute timer so we don't trigger rate limits.
;Debug stuff
global DEBUG := 1 ; 1=on 0=off
global logPath := A_ScriptDir . "\debug.log"
global SizeLimit := 300 * 1024 * 1024  ; logs archive at 300MB
global ratioList := ["Default","9:16","2:3","3:4","4:5","1:1","5:4","4:3","3:2","16:9","21:9"]
global ratioListExt := ["Default","1:8","1:4","9:16","2:3","3:4","4:5","1:1","5:4","4:3","3:2","16:9","21:9","4:1","8:1"]
ver := "7.13"
; } These don't change in program.

; Variables {
global useCurl := 1 ; Setting this to 0 uses a HTTPS mode that freezes the GUI! Using curl is recommended.
global CurrentMonitorIndex := 1
global UIW:=900 ; UI width
global imgw := 300
global imgh := 200
global TotalBatchCost := 0.0
global ImageTaskMap := []
global CurrentPath := ""
global IsBatchRunning := false
global CurrentBatchIndex := 0
global Data := ""
global LastFPress := 0
global NextImageID := 1
global PendingTasks := 0
global CurlTimers := Map()
global ActiveStreams := Map()
; GUI Controls
global MyGui, Tab, LV_Images, Pic_Preview, LV_Tasks, Btn_Gen, Btn_Add, TotalCostDisplay, Radio_Immediate, Radio_Batch, Btn_load, Btn_save, Btn_Test, Btn_Run, Prog_Bar
global ed, neg, tier, ratio, fmt, popCost, batView, Btn_ClearBatches, batBar, ModelLog, OutT, OutB, OutL, OutR
; }

if !DirExist(OutputDir)
    DirCreate(OutputDir)

; --- UI SETUP ---
MyGui := Gui("+Resize", "Gemini 2026 Pro Editor")
MyGui.OnEvent("DropFiles", Gui_DropFiles)
MyGui.SetFont("s10", "Segoe UI")

Tab := MyGui.Add("Tab3", "w" . UIW+40 . " h500", ["Create", "Batches","Help"])

Tab.UseTab(1)
LV_Images := MyGui.Add("ListView", "w" . imgw . " h" . imgh, ["#", "MBs", "tasks", "Image", "Path"])
;LV_Images.ModifyCol(2, 0)
LV_Images.OnEvent("Click", ImageListClick)
LV_Images.OnEvent("ItemFocus", ImageListClick)
LV_Images.OnEvent("DoubleClick", ImageListDoubleClick)

Pic_Preview := MyGui.Add("Pic", "x+10 yp w" . imgw . " h" . imgh . " +Border +Center ", "")
;DllCall("uxtheme\SetWindowTheme", "Ptr", OutT.Hwnd, "Str", "", "Str", "")
; -E0x20000 Vertical -Border
global OutT := MyGui.Add("Progress", "BackgroundGreen cRed hidden", 100)
global OutB := MyGui.Add("Progress", "BackgroundGreen cRed hidden", 100)
global OutL := MyGui.Add("Progress", "BackgroundGreen cRed hidden", 100)
global OutR := MyGui.Add("Progress", "BackgroundGreen cRed hidden", 100) 
global outsz:=1
SetTimer(UpdateRatioBars, 100)

MyGui.SetFont("s10 norm")
LV_Tasks := MyGui.Add("ListView", "x30 y250 w" . imgw*2+10 . " h140", ["Img", "Agent", "Res", "Ratio", "Status", "Cost ($)", "Prompt", "TaskID", "TaskIdx"])
LV_Tasks.ModifyCol(8, 0)
LV_Tasks.ModifyCol(9, 0)
;LV_Tasks.OnEvent("Click", TaskListClick)
LV_Tasks.OnEvent("Click", (*) => UpdateButtonStates())
LV_Tasks.OnEvent("ItemSelect", (*) => UpdateButtonStates())
LV_Tasks.OnEvent("ItemFocus", (*) => UpdateButtonStates())
;LV_Tasks.OnEvent("DoubleClick", ShowTaskForm)

MyGui.SetFont("s12 bold")
Btn_Gen := MyGui.Add("Button", " w100 h30", "New Image")
Btn_Add := MyGui.Add("Button", "x+10 yp w110 h30 Disabled", "Add Task")
Btn_Add.OnEvent("Click", CreateNewTask)
Btn_Gen.OnEvent("Click", AddNullImage)
TotalCostDisplay := MyGui.Add("Text", "x+10 yp w120", "Total: $0.0000")
Radio_Immediate := MyGui.Add("Radio", "x+10 yp Checked", "Immediate")
Radio_Immediate.OnEvent("Click", RefreshAllCosts) ; Add this trigger
Radio_Batch := MyGui.Add("Radio", "x+20", "Batch")
Radio_Batch.OnEvent("Click", RefreshAllCosts) ; Add this trigger

Btn_load := MyGui.Add("Button", "x30 yp+40 w100 h30", "Load CSV")
Btn_load.OnEvent("Click", LoadCSV)
Btn_save := MyGui.Add("Button", "x+10 yp w100 h30 disabled", "Save CSV")
Btn_save.OnEvent("Click", SaveCSV)
Btn_Test := MyGui.Add("Button", "x+10 yp w100 h30", "Test API Key")
Btn_Test.OnEvent("Click", TestAPIConnection)
Btn_Run := MyGui.Add("Button", "x+10 yp w175 h30 Disabled", "RUN IMMEDIATE")
Btn_Run.OnEvent("Click", StartBatch)

Prog_Bar := MyGui.Add("Progress", "x20 y480 w" . imgw*2+20 . " h15 cYellow", 0)

; task gui here ;
MyGui.Add("Text","x" . imgw*2+50 . " y40", "Positive Prompt:")
ed := MyGui.Add("Edit", "w290 r7 disabled" )
ed.OnEvent("Change", AutoSaveTask)

MyGui.Add("Text",, "Negative Prompt:")
neg := MyGui.Add("Edit", "w290 r3 disabled")
neg.OnEvent("Change", AutoSaveTask)

MyGui.Add("Text", "w100", "Tier:")
tier := MyGui.Add("DropDownList", "x+10 disabled", ["Nano Flash 1K","Nano Pro 1K", "Nano Pro 2K", "Nano Pro 4K","Nano 2 1K","Nano 2 2K","Nano 2 4K","Imagen 2K","Imagen Ultra 2K"])
tier.OnEvent("Change", AutoSaveTask)

MyGui.Add("Text", "x" . imgw*2+50 . " y+10 ", "Aspect Ratio:")
ratio := MyGui.Add("DropDownList", "x+10 disabled", ratioList)
ratio.OnEvent("Change", AutoSaveTask)

MyGui.Add("Text", "x" . imgw*2+50 . " y+10 w100", "Output:")
fmt := MyGui.Add("DropDownList", "x+10 disabled", ["JPG", "PNG"])
fmt.OnEvent("Change", AutoSaveTask)

MyGui.Add("Text", "x" . imgw*2+50 . " y+10 w100", "Cost:")
popCost := MyGui.Add("Text", "x+10 w145", "$0.00")
    
    

Tab.UseTab(2)
;batView := MyGui.Add("ListView", "x20 y50 w" . imgw*2+40 . " h380 Grid", ["File", "Agent", "Res", "Status", "Load", "JobID"])
batView := MyGui.Add("ListView", "x20 y50 w" . UIW+20 . " h380 Grid", ["JobID", "Status", "Submitted", "Progress"])
Btn_ClearBatches := MyGui.Add("Button", "x20 yp+390 w150 h30", "Clear Finished")
Btn_ClearBatches.OnEvent("Click", ClearFinishedJobs)
batBar := MyGui.Add("Progress", "x20 y480 w" . UIW+20 . " h15 cGreen", 0)

Tab.UseTab(3)
HelpDisp := MyGui.Add("Text", "w" . imgw*2+40 . " h600", "DBnano v" . ver . "`n`nNano Banana (Flash, pro and 2) can edit or generate multiple images and do batch.`nImagen is text to image only.`n`nTask flow:`nDrag and drop images on the app or click Generate for a new image.`nSelect the images you want with Shift or CTRL.`nClick Add Task, fill out the right side details (like prompt and ratio)`nChoose Immediate or Batch`nClick Run!`n`nImmediate mode should make the image in a few minutes.`nBatch mode adds everything to a waitlist, see the second tab for progress.`n`nYou can save/load your prompt settings as CSV.`n`nCosts are based on 2026 data, subject to change")

Tab.UseTab()
global ModelLog := MyGui.Add("Edit", "xm y500 w" . UIW+40 . " r5 +ReadOnly +vModelLog", "")
MyGui.Show()

SetTimer(LoadExistingJobs, -500)

if (useCurl) {
  if (FileExist(A_ScriptDir . "\curl.exe") || FileExist(A_WinDir . "\System32\curl.exe")) {
    ModelLogMsg("Found curl.exe using curl mode.")
  } else {
    useCurl := 0
    ModelLogMsg("Cannot find curl.exe using standard mode.  The GUI will freeze when doing HTTPS.")
  }
} else {
 ModelLogMsg("Using standard mode. The GUI will freeze when doing HTTPS.")
}
ModelLogMsg("To start, drag and drop an image in the app or click 'New Image'.")
ModelLogMsg("Double click the image entry to change it.")

CreateNewTask(*) {
    row := 0
    selectedPaths := []
    selectedIDs := []

    Loop {
        row := LV_Images.GetNext(row, "")
        if not row
            break
        selectedIDs.Push(LV_Images.GetText(row, 1))
        selectedPaths.Push(LV_Images.GetText(row, 5))
    }

    if (selectedPaths.Length == 0) {
        MsgBox "Please select one or more images first!"
        return
    }

    ; 2. Create the combined strings
    ; ID string for the GUI (e.g., "1, 2, 3")
    combinedIDs := ""
    for id in selectedIDs
        combinedIDs .= (A_Index == 1 ? "" : ",") . id

    ;tooltip combinedIDs

    ; Path string for the logic (e.g., "path1|path2|path3")
    combinedPaths := ""
    for p in selectedPaths
        combinedPaths .= (A_Index == 1 ? "" : "|") . p

    ; 3. Grab the ratio from the first image only
    masterRatio := "1:1"
    currentImgPath := selectedPaths[1]
    if (currentImgPath != "" && currentImgPath != "<GENERATE>") {
        try {
            temp := Gui()
            pic := temp.Add("Pic",, currentImgPath)
            pic.GetPos(,, &iw, &ih)
            temp.Destroy()
            masterRatio := GetClosestRatio(iw, ih)
        }
    }
    ; 4. Create the single task object
    newTask := {
        ID: selectedIDs[1],        ; id
        IDs: combinedIDs,        ; This shows "1, 2, 3" in the Task List
        Prompt: proVal, 
        NegativePrompt: negVal,
        Agent: "Nano Flash",
        Size: "1K",
        Ratio: masterRatio,     ; Pulled from image #1
        Format: "JPG",
        Status: "Pending",
        SourcePath: combinedPaths, ; The pipe-separated string for processing
        Cost: CalculateCost("Nano Flash", "1K")
    }

    ; 5. Add to the array and refresh
    ImageTaskMap.Push(newTask)

    ; 6. Update task counts for all selected images
    for id in selectedIDs {
        Loop LV_Images.GetCount() {
            if (LV_Images.GetText(A_Index, 1) == id) {
                taskCount := 0
                for t in ImageTaskMap {
                    if (IsIDInMergedID(id, t.IDs)) {
                        taskCount++
                    }
                }
                LV_Images.Modify(A_Index, "", , , taskCount)
                break
            }
        }
    }

    RefreshTaskTable()
    UpdateTotalDisplay()
   
    ; Find the newly added row and select it to enable the side panel
    ;Loop LV_Tasks.GetCount() {
        ;if (LV_Tasks.GetText(A_Index, 1) == imgID) {
            ;LV_Tasks.Modify(A_Index, "Select Focus")
        ;}
    ;}
}

SaveCSV(*) {
    savePath := FileSelect("S16", A_ScriptDir, "Save Task Configuration", "CSV (*.csv)")
    if (savePath == "")
        return

    if !RegExMatch(savePath, "i)\.csv$")
        savePath .= ".csv"

    try {
        fileObj := FileOpen(savePath, "w", "UTF-8")

        Loop LV_Images.GetCount() {
            imgID := LV_Images.GetText(A_Index, 1)
            filePath := LV_Images.GetText(A_Index, 5)
            fileObj.WriteLine("img," . imgID . "," . filePath)
        }
        for task in ImageTaskMap {
            ; Cleaning BOTH prompts of commas to prevent column shifting
            cleanPrompt := StrReplace(task.Prompt, ",", "¢")
            cleanNeg    := StrReplace(task.NegativePrompt, ",", "¢")
            cleanPrompt := StrReplace(StrReplace(cleanPrompt, "`r", " "), "`n", " ") ; no lf cr
            cleanNeg := StrReplace(StrReplace(cleanNeg, "`r", " "), "`n", " ") ; no lf cr

            ; New structure: tsk, parentIdx, size, agent, ratio, prompt, negPrompt, format
            line := "tsk," . task.ID . "," . task.Size . "," . task.Agent . "," . task.Ratio . "," . cleanPrompt . "," . cleanNeg . "," . task.Format
            fileObj.WriteLine(line)
        }
        fileObj.Close()
        ModelLogMsg("Configuration saved with prompt sanitization.")
    } catch Error as e {
        MsgBox "Save failed: " . e.Message
    }
}

LoadCSV(*) {
    loadPath := FileSelect(3, A_ScriptDir, "Open Task Configuration", "CSV (*.csv)")
    if (loadPath == "")
        return

    global ImageTaskMap := []
    global NextImageID := 1
    LV_Images.Delete()
    LV_Tasks.Delete()
    tempImgMap := Map() ; Link CSV index to path

    try {
        Loop Read, loadPath {
            parts := StrSplit(A_LoopReadLine, ",")
            if (parts.Length < 3)
               continue

            if (parts[1] == "img") {
                idx := parts[2]
                path := parts[3]
                if (path == "<GENERATE>" || FileExist(path)) {
                    if (path == "<GENERATE>") {
                        fn := "GENERATE"
                        sizeMB := "0.00"
                    } else {
                        SplitPath path, &fn
                        sizeMB := Format("{:.2f}", FileGetSize(path) / 1024 / 1024)
                    }
                    ix := String(NextImageID++)
                    LV_Images.Add(, ix, sizeMB, 0, fn, path)
                    tempImgMap[idx] := ix
                }
            }
            else if (parts[1] == "tsk") {
                parentIdx := parts[2]
                if tempImgMap.Has(parentIdx) {
                    currentIx := tempImgMap[parentIdx]

                    newTask := {
                        ID: currentIx,
                        IDs: currentIx,
                        Size: parts[3],
                        Agent: parts[4],
                        Ratio: parts[5],
                        Prompt: StrReplace(parts[6], "¢", ","),
                        NegativePrompt: StrReplace(parts[7], "¢", ","),
                        Format: parts[8],
                        Status: "Pending",
                        SourcePath: "", ; Will be set below
                        Cost: CalculateCost(parts[4], parts[3]),
                        Mode: Radio_Batch.Value ? "Batch" : "Immediate"
                    }

                    ; Retrieve the SourcePath from the ListView using the current image index
                    Loop LV_Images.GetCount() {
                        if (LV_Images.GetText(A_Index, 1) == currentIx) {
                            newTask.SourcePath := LV_Images.GetText(A_Index, 5)
                            break
                        }
                    }

                    ImageTaskMap.Push(newTask)
                }
            }
        }
        ; Update task counts in ListView
        Loop LV_Images.GetCount() {
            id := LV_Images.GetText(A_Index, 1)
            taskCount := 0
            for t in ImageTaskMap {
                if (IsIDInMergedID(id, t.IDs)) {
                    taskCount++
                }
            }
            LV_Images.Modify(A_Index, "", , , taskCount)
        }
        ; UI Refresh logic
        if (LV_Images.GetCount() > 0) {
            LV_Images.Modify(1, "Select Focus")
            ImageListClick(LV_Images, 1)
        }
        UpdateTotalDisplay()
        RefreshTaskTable()
        UpdateButtonStates()
    } catch Error as e {
        MsgBox "Load failed: " . e.Message
    }
}

; --- 1. THE SCALING FIX ---
UpdatePreview(ImgPath) {
    if (ImgPath == "<GENERATE>") {
        ;Pic_Preview.Value := ""
        ;return
      pw:=300
      ph:=200
      Pic_Preview.Value := "*w" pw*(A_ScreenDPI/96) " *h" ph*(A_ScreenDPI/96)
    }
    if (ImgPath == "" || !FileExist(ImgPath)) {
        return
    }
    temp := Gui()
    pic := temp.Add("Pic",, ImgPath)
    pic.GetPos(,, &iw, &ih)
    temp.Destroy()
    if (iw/ih > imgw/imgh) {
        Pic_Preview.Value := "*w" . imgw*(A_ScreenDPI/96) . " *h-1 " . ImgPath
    } else {
        Pic_Preview.Value := "*w-1 *h" . imgh*(A_ScreenDPI/96) . " " . ImgPath
    }
}

RefreshAllCosts(*) {
    Btn_Run.Text := Radio_Batch.Value ? "RUN BATCH" : "RUN IMMEDIATE"

    ; Loop through every task
    for t in ImageTaskMap {
        ; Re-calculate the cost based on the current toggle state
        t.Cost := CalculateCost(t.Agent, t.Size)
        ; Update the mode of the task to match the new toggle
        t.Mode := Radio_Batch.Value ? "Batch" : "Immediate"
    }

    ; Update the "Total: $0.0000" text
    UpdateTotalDisplay()
    ; Update the ListView rows to show the new prices
    RefreshTaskTable()
    UpdateButtonStates()
}

LoadExistingJobs() {
    jobFile := A_ScriptDir "\jobs.txt"
    if !FileExist(jobFile)
        return

    jobCount:=0
    Loop Read, jobFile {
        if (A_LoopReadLine == "")
         continue

        ;batView.Add(, A_LoopReadLine, "Checking...", A_LoopFileTimeModified, "0%")
        batView.Add(, A_LoopReadLine, "Checking...", "Prior Session", "0%")
        foundList .= "`n  - " . A_LoopReadLine
        jobCount++
    }
    batView.ModifyCol()

    if (jobCount == 0) {
        ModelLogMsg("jobs.txt is empty. No jobs found.")
        return
    }

    ; Success: Log the list for the user to see in the status window
    ModelLogMsg("Found Jobs:" . foundList)

    global NextCheckTime := A_TickCount + + CheckInterval
    SetTimer(UpdateMonitorProgress, 1000)
}

CalculateCost(agent, res := "1K") {
 full := agent . " " . res
 base := 0.134 ; pro 1K & pro 2k
 nano := 1
 if (InStr(full, "Imagen 2K")) {
  base:= 0.04
  nano:=0
 }
 if (InStr(full, "Ultra 2K")) {
  base:= 0.06
  nano:=0
 }
 if (InStr(full, "Pro 4K")) {
   base:= 0.24
 }
 if (InStr(full, "2 1K")) {
   base:= 0.067
 }
 if (InStr(full, "2 2K")) {
   base:= 0.101
 }
 if (InStr(full, "2 4K")) {
   base:= 0.151
 }
 if (InStr(full, "Flash")) {
  base:= 0.039
 }
 ; Apply 50% discount if Batch Mode is selected
 return Radio_Batch.Value && nano ? (base * 0.5) : base
}

ProcessMergedSelection(Prompt, FullTierName, EntryGui) {
    ; Extract Agent and Size from "Nano Pro 4k" -> Agent: "Nano Pro", Size: "4k"
    if RegExMatch(FullTierName, "(.*)\s(\d+K)", &match) {
        Agent := match[1]
        Size := match[2]
        ;SubmitTaskWithExtras(Data)
    }
}

UpdateTotalDisplay() {
    global TotalBatchCost := 0.0
    ;if (Radio_Upscale.Value) {
    ;    TotalBatchCost := LV_Images.GetCount() * 0.003
    ;} else {
        for t in ImageTaskMap {
            TotalBatchCost += t.Cost
        }
    ;}
    TotalCostDisplay.Value := "Total: $" . Format("{:.4f}", TotalBatchCost)
}

RefreshTaskTable() {
    LV_Tasks.Delete()

    ; Loop through the Array
    for i, t in ImageTaskMap {
        ; Column 1 clearly shows which Image ID this task belongs to
        ; Col 5 is Status, Col 6 is Cost, Col 7 is Prompt, Col 8 is TaskID, Col 9 is TaskIdx
        LV_Tasks.Add(, t.IDs, t.Agent, t.Size, t.Ratio, t.Status, Format("{:.3f}", t.Cost), t.Prompt, t.ID, i)
    }
    LV_Tasks.ModifyCol()
    LV_Tasks.ModifyCol(8, 0) ; Hide TaskID
    LV_Tasks.ModifyCol(9, 0) ; Hide TaskIdx
}

AsyncSubmitBatchJob(fileUri, selectedModel) {
    global useCurl, API_KEY, CurlTimers
    apiUrl := "https://generativelanguage.googleapis.com/v1beta/models/" . selectedModel . ":batchGenerateContent?key=" . API_KEY

    RegExMatch(fileUri, "files/[^/`"]+", &match)
    fileId := match ? match[0] : fileUri
    payload := '{ "batch": { "input_config": { "file_name": "' . fileId . '" } } }'

    if (useCurl) {
        resFile := A_ScriptDir . "\gemini_batch_sub_" . A_TickCount . ".json"
        payloadFile := A_ScriptDir . "\gemini_batch_sub_pay_" . A_TickCount . ".json"
        FileAppend(payload, payloadFile, "UTF-8-RAW")

        curlCmd := 'curl -s -X POST "' . apiUrl . '" -H "Content-Type: application/json" -d "@' . payloadFile . '" -o "' . resFile . '"'
        Run(curlCmd, , "Hide", &pid)
        cb := ProcessBatchSubCurl.Bind(pid, resFile, payloadFile)
        CurlTimers[pid] := cb
        SetTimer(cb, 100)
    } else {
        whr := ComObject("WinHttp.WinHttpRequest.5.1")
        whr.SetTimeouts(30000, 60000, 600000, 600000)
        whr.Open("POST", apiUrl, true) ; Async
        whr.SetRequestHeader("Content-Type", "application/json")
        whr.Send(payload)
        SetTimer(CheckWinHttpSub.Bind(whr), 100)
    }
}

ProcessBatchSubCurl(pid, resFile, payloadFile) {
    if ProcessExist(pid)
        return

    global CurlTimers
    if CurlTimers.Has(pid) {
        SetTimer(CurlTimers[pid], 0)
        CurlTimers.Delete(pid)
    }

    resText := ""
    if FileExist(resFile) {
        resText := FileRead(resFile)
        FileDelete(resFile)
    }
    if FileExist(payloadFile) {
        FileDelete(payloadFile)
    }

    if (resText == "" || InStr(resText, '"error"')) {
        BatchError("Batch Submission Failed: " . resText)
    } else {
        FinishBatchSubmission(resText)
    }
}

CheckWinHttpSub(whr) {
    if (whr.ReadyState != 4)
        return

    SetTimer(, 0)
    if (whr.Status == 200 && !InStr(whr.ResponseText, '"error"')) {
        FinishBatchSubmission(whr.ResponseText)
    } else {
        BatchError("Batch Submission Failed (" . whr.Status . "): " . whr.ResponseText)
    }
}

FinishBatchSubmission(responseText) {
    jobID := JSON_Get(responseText, "name")
    if (jobID == "") {
         BatchError("Failed to parse Job ID from response: " . responseText)
         return
    }

    FileAppend(jobID . "`n", A_ScriptDir "\jobs.txt")

    batView.Add(, jobID, "Submitted", FormatTime(, "HH:mm:ss"), "0%")
    batView.ModifyCol()
    global NextCheckTime := A_TickCount + CheckInterval
    SetTimer(UpdateMonitorProgress, 1000)

    ModelLogMsg("Batch Submitted: " . jobID)
    SetLoadingState(false)
    Prog_Bar.Value := 0
    ToggleUI(true)
}

BatchError(msg) {
    global DEBUG
    if (DEBUG)
        LogMessage("BATCH CRITICAL ERROR: " . msg)

    SetLoadingState(false)
    Prog_Bar.Value := 0
    ModelLogMsg("Batch Failed: " . msg)
    ToggleUI(true)
}


CreateBatchFile(TaskArray, selectedModel) {
    batchPath := A_ScriptDir "\batch_job.jsonl"
     ;if FileExist(batchPath)
     ;   FileDelete(batchPath)

    fileObj := FileOpen(batchPath, "w", "UTF-8-RAW")
    modelPath := "models/" . selectedModel  ;url stuff

    for task in TaskArray {
        ; Use the specific SourcePath saved with this task
        currentTaskPath := task.SourcePath
        fn := StrReplace(currentTaskPath, "\", "_")

        ; Pass the specific task's path to the payload creator
        payload := CreateJsonPayload(task, currentTaskPath)
        if (payload=="err") {
         return
        }
        payload := Trim(payload)
        payload := RegExReplace(payload, "[\r\n\t]+", " ")
        payload := RegExReplace(payload, "\s+", " ")
        payload := Trim(payload)

        line := '{"custom_id": "' . fn . '", "request": {"model": "' . modelPath . '", ' . SubStr(payload, 2) . '}'
        fileObj.WriteLine(line)
    }
    fileObj.Close()
    return batchPath
}

AddNullImage(*) {
    global NextImageID
    ix := String(NextImageID++)
    LV_Images.Add(, ix, "0.00", 0, "GENERATE", "<GENERATE>")
    LV_Images.Modify(LV_Images.GetCount(), "Select Focus")
    ImageListClick(LV_Images, LV_Images.GetCount())
    UpdateTotalDisplay()
    UpdateButtonStates()
    ModelLogMsg("Great, now add a task for the selected generated image!")
}

Gui_DropFiles(GuiObj, GuiCtrlObj, FileArray, X, Y) {
    global NextImageID
    for i, file in FileArray {
        SplitPath file, &fn
        ix := String(NextImageID++)
        sizeMB := Format("{:.2f}", FileGetSize(file) / 1024 / 1024)
        LV_Images.Add(, ix, sizeMB, 0, fn, file)
    }
    LV_Images.ModifyCol(1, "AutoHdr")
    if (LV_Images.GetCount() > 0) {
        LV_Images.Modify(1, "Select Focus") ; Select the first row
        ImageListClick(LV_Images, 1)        ; Trigger the preview and task list logic
    }
    LV_Images.ModifyCol()
    UpdateTotalDisplay()
    UpdateButtonStates()
    ModelLogMsg("Great, now add a task for the selected images!")
}

UpdateButtonStates() {
    global ed, neg, tier, ratio, fmt, popCost, ImageTaskMap
    selectedTaskRow := LV_Tasks.GetNext(0, "Focused")
    taskCount := LV_Tasks.GetCount()
    imgCount  := LV_Images.GetCount()

    ; --- GLOBAL BUTTON LOGIC ---
    Btn_Add.Enabled := (imgCount > 0)
    btn_Save.Enabled := (taskCount > 0)
    
    ; Run Batch: Must be uniform AND non-Imagen
    btn_Run.Enabled := AreAgentsUniformAndBatchable()
    
    ; --- 2. SIDE PANEL LOGIC ---
    if (selectedTaskRow > 0) {
        ; A task is selected: Enable and Populate
        ed.Enabled := true
        neg.Enabled := true
        tier.Enabled := true
        ratio.Enabled := true
        fmt.Enabled := true
        
        localIdx := Integer(LV_Tasks.GetText(selectedTaskRow, 9))
        
        if localIdx > 0 && localIdx <= ImageTaskMap.Length {
            ;ModelLogMsg("Loading task #" . localIdx)
            task := ImageTaskMap[localIdx]
            ed.Value := task.Prompt
            neg.Value := task.NegativePrompt
            tier.Text := task.Agent . " " . task.Size

            ; Conditional ratio list for 3.1 Flash
            ;tooltip task.Agent
            if InStr(task.Agent, "Nano 2") {
                ratio.Delete()
                ratio.Add(ratioListExt)
            } else {
                ratio.Delete()
                ratio.Add(ratioList)
            }

            ratio.Text := task.Ratio
            fmt.Text := task.Format
            popCost.Value := "$" . Format("{:.3f}", task.Cost)
            
            ; Imagen doesn't support negative prompts, so disable if needed
            neg.Enabled := !InStr(task.Agent, "Imagen")
        }
    } else {
        ; NOTHING is selected: Disable and Clear the side panel
        ed.Value := ""
        neg.Value := ""
        ed.Enabled := false
        neg.Enabled := false
        tier.Enabled := false
        ratio.Enabled := false
        fmt.Enabled := false
        popCost.Value := "$0.00"
    }
    UpdateRatioOutline()
}

ImageListClick(LV, RowNum) {
    if (RowNum <= 0 || RowNum > LV.GetCount())
        return ;

    ; Reset button to "Add Task" when switching images
    Btn_Add.Text := "Add Task"

    try {
        fullPath := LV.GetText(RowNum, 5) ; // Column 5 is Full Path
        if (fullPath != "") {
            if (fullPath == "<GENERATE>" || FileExist(fullPath)) {
                global CurrentPath := fullPath
                UpdatePreview(fullPath) ;
                RefreshTaskTable() ;
            }
        }
    } catch {
        return
    }
    UpdateButtonStates()
}

ImageListDoubleClick(LV, RowNum) {
    if (RowNum <= 0 || RowNum > LV.GetCount())
        return

    newPath := FileSelect(3, A_ScriptDir, "Select New Image", "Images (*.jpg; *.png; *.jpeg)")
    if (newPath == "")
        return

    SplitPath newPath, &fn
    sizeMB := Format("{:.2f}", FileGetSize(newPath) / 1024 / 1024)
    imgID := LV.GetText(RowNum, 1)

    ; Update ListView: Col 2 (MBs), Col 4 (Image), Col 5 (Path)
    LV.Modify(RowNum, "", , sizeMB, , fn, newPath)

    ; Update SourcePath in ImageTaskMap for ALL tasks involving this image
    for t in ImageTaskMap {
        if (IsIDInMergedID(imgID, t.IDs)) {
             newPaths := ""
             ids := StrSplit(t.IDs, ",")
             for id in ids {
                 path := ""
                 Loop LV_Images.GetCount() {
                     if (LV_Images.GetText(A_Index, 1) == id) {
                         path := LV_Images.GetText(A_Index, 5)
                         break
                     }
                 }
                 newPaths .= (newPaths == "" ? "" : "|") . path
             }
             t.SourcePath := newPaths
        }
    }

    ; Trigger click logic to update preview and paths
    ImageListClick(LV, RowNum)
}

ToggleUI(Enable := true) {
    Btn_Run.Enabled := Enable
    Btn_Add.Enabled := Enable
    Radio_Immediate.Enabled := Enable
    Radio_Batch.Enabled := Enable
    ;Status_Text.Value := Enable ? "Ready" : "Processing... Please wait."
}

StartBatch(*) {
    global MODEL1, MODEL2, MODEL3, MODEL4, MODEL5, ImageTaskMap, Radio_Batch, LV_Tasks, ModelLog, Prog_Bar, DEBUG

    firstAgent := ""
    isMixed := false
    selectedBatchModel := ""

    if (LV_Tasks.GetCount() == 0) {
        MsgBox "No tasks to run!"
        return
    }
    ;fnf:="xxx"
    if (Radio_Batch.Value) {
        firstSize := ""
        for taskObj in ImageTaskMap {
            ;ModelLogMsg("[]: " . taskObj.IDs )
            ;FileExist
            if (firstAgent == "") {
                firstAgent := taskObj.Agent
                firstSize := taskObj.Size
            } else if (taskObj.Agent != firstAgent || (InStr(firstAgent, "Imagen") && taskObj.Size != firstSize)) {
                isMixed := true
                break
            }
        }
        ;if (fnf) {
        ;  ModelLogMsg("[Error] File not found: " . fnf )
        ;  return
        ;}
        if (isMixed) {
            result := MsgBox("Warning: Your batch contains a mix of models (Flash, Pro, or Imagen).`n`nGoogle Batch API requires all tasks in a single job to use the SAME model.")
            return
        }

        if (InStr(firstAgent, "Imagen")) {
            MsgBox("Imagen does not currently support Batch Mode. Please use Immediate mode.")
            return
        }
    }
    ; --- END OF CHECK ---

    ToggleUI(false)

    if (Radio_Batch.Value) {
        ModelLogMsg("Starting Batch Upload...")
        SetLoadingState(true)

        try {
            if InStr(firstAgent, "Nano 2")
                selectedBatchModel := MODEL5
            else if InStr(firstAgent, "Flash")
                selectedBatchModel := MODEL1
            else if InStr(firstAgent, "Imagen") {
                if (InStr(firstAgent, "Ultra"))
                    selectedBatchModel := MODEL4
                else
                    selectedBatchModel := MODEL3
            } else
                selectedBatchModel := MODEL2

            batchPath := CreateBatchFile(ImageTaskMap, selectedBatchModel)
            if (!batchPath) {
              BatchError("Batch not created!")
              return
            }
            LogMessage("BATCH START: File created at " . batchPath)

            AsyncUploadBatchFile(batchPath, selectedBatchModel)
        } catch Error as e {
            BatchError(e.Message)
        }
    } else {
        global IsBatchRunning := true
        global CurrentBatchIndex := 0
        Prog_Bar.Value := 0
        ModelLogMsg("Starting Immediate Queue...")
        SetTimer ProcessNextTask, 5000
    }
}

FakeProgress() {
    Prog_Bar.Value := Random(10, 90)
}

; Function to start/stop the jumping
SetLoadingState(active) {
    if (active) {
        SetTimer(FakeProgress, 200) ; Jump every 200ms
    } else {
        SetTimer(FakeProgress, 0)   ; Stop jumping
        Prog_Bar.Value := 0
    }
}

ProcessNextTask() {
    global useCurl, PendingTasks
    global CurrentBatchIndex, ImageTaskMap

    if (PendingTasks > 0)
        return ; Wait for previous task to finish (Rate Limit Safety)

    TotalTasks := LV_Tasks.GetCount()

    if (CurrentBatchIndex >= TotalTasks) {
        SetTimer(ProcessNextTask, 0) ; // Stop the timer
        global IsBatchRunning := false
        if (!useCurl || PendingTasks == 0)
            ToggleUI(true)               ; // Re-enable buttons
        return
    }

    CurrentBatchIndex++ ; // Move to the next row

    ; // Get the Task identifiers from the UI
    localIdx := Integer(LV_Tasks.GetText(CurrentBatchIndex, 9))

    found := localIdx > 0 && localIdx <= ImageTaskMap.Length

    ; // Retrieve the object
    if (found) {
        try {
            targetTask := ImageTaskMap[localIdx]
            RunGeminiTask(targetTask.SourcePath, targetTask, CurrentBatchIndex)
        } catch Error as e {
            ;if you get this, try disabling the try/catch
            ModelLogMsg("Task mapping error: " e.Message . " " . localIdx)
        }
    }
}

RunGeminiTask(fullPath, taskObj, batchIdx) {
    global API_KEY, MODEL1, MODEL2, MODEL3, MODEL4, MODEL5, hurl, useCurl, PendingTasks, CurlTimers, DEBUG, OutputDir, ModelLog

   ;if (!FileExist(fullPath)) { ;xxx
   ;  ModelLogMsg("[Error] File not found: " . fullPath )
   ;  return
   ;}
   paths := StrSplit(fullPath, "|")
   for eachPath in paths {
        if (eachPath != "<GENERATE>" && !FileExist(eachPath)) {
            ModelLogMsg("[Error] File not found: " . eachPath)
            return ; Stop the task if any single file is missing
        }
    }

    MODEL_ID := ""
    payload := ""
    apiUrl := ""

    ; // Extract variables from the task object
    agent := taskObj.Agent
    size  := taskObj.Size
    prompt := taskObj.Prompt

    ; // Get file details for the filename/extension logic
    nameNoExt := ""
    nameWithExt := ""
    dir := ""
    ext := ""
    if (fullPath == "<GENERATE>") {
        nameNoExt := "Generated"
    } else if InStr(fullPath, "|") {
        nameNoExt := "Merged_" . StrReplace(taskObj.ID, "+", "_")
    } else {
        SplitPath fullPath, &nameWithExt, &dir, &ext, &nameNoExt
    }


    if InStr(agent, "Nano 2")
        MODEL_ID := MODEL5
    else if InStr(agent, "Flash")
        MODEL_ID := MODEL1
    else if InStr(agent, "Imagen") {
        if (InStr(agent, "Ultra"))
            MODEL_ID := MODEL4
        else
            MODEL_ID := MODEL3
    } else
        MODEL_ID := MODEL2
    if (useCurl) {
        payloadFile := A_ScriptDir . "\gemini_pay_" . A_TickCount . "_" . batchIdx . ".json"
        responseFile := A_ScriptDir . "\gemini_res_" . A_TickCount . "_" . batchIdx . ".json"

        authHeader := ""
         if InStr(agent, "Imagen") {
            apiUrl := hurl . MODEL_ID . ":predict?key=" . API_KEY
         } else {
            apiUrl := hurl . MODEL_ID . ":streamGenerateContent?key=" . API_KEY
         }
         payload := CreateJsonPayload(taskObj, fullPath)

        if FileExist(payloadFile)
          FileDelete(payloadFile)
          
        FileAppend(payload, payloadFile, "UTF-8-RAW")

        idxLog := "Task " . batchIdx
        LogMessage(idxLog . " API URL: " . apiUrl)
        LogMessage(idxLog . " Payload: " . payload)

        curlLogFile := A_ScriptDir . "\gemini_curl_err_" . A_TickCount . "_" . batchIdx . ".log"
        curlCmd := 'curl -s -S -N -X POST "' . apiUrl . '" -H "Content-Type: application/json"' . authHeader . ' -d "@' . payloadFile . '" -o "' . responseFile . '" 2> "' . curlLogFile . '"'
        LogMessage(idxLog . " Curl Command: " . curlCmd)
        Run(curlCmd, , "Hide", &pid)
        global PendingTasks += 1
        CurlTimers[pid] := CheckCurlProgress.Bind(pid, responseFile, payloadFile, batchIdx, nameNoExt)
        SetTimer(CurlTimers[pid], 200)
        ModelLogMsg("[curl] " . idxLog . " started (PID: " . pid . ")")
        return
    }



    try {
        payload := CreateJsonPayload(taskObj, fullPath)

        ; --- DEBUG: LOG WHAT IS SENT ---
        if (DEBUG) {
            LogMessage("--- SENDING TO API ---`n" . payload . "`n")
        }
        idxLog := "Task " . batchIdx
        ModelLogMsg("Info [" . idxLog . "]: " . MODEL_ID . " " . taskObj.Ratio . " " . taskObj.Size . " " . nameNoExt)

        whr := ComObject("WinHttp.WinHttpRequest.5.1")
        ;SetTimeouts(resolve, connect, send, receive) in milliseconds
        whr.SetTimeouts(30000, 60000, 600000, 600000)

            if InStr(agent, "Imagen") {
                apiUrl := hurl . MODEL_ID . ":predict?key=" . API_KEY
            } else {
                apiUrl := hurl . MODEL_ID . ":generateContent?key=" . API_KEY
            }
            whr.Open("POST", apiUrl, false)

        whr.SetRequestHeader("Content-Type", "application/json")
        whr.Send(payload)

        ; --- DEBUG: LOG WHAT IS RECEIVED ---
        if (DEBUG) {
            LogMessage("--- RECEIVED FROM API ---`nStatus: " . whr.Status . "`nResponse: " . whr.ResponseText . "`n")
        }

if (whr.Status == 200) {
    responseText := whr.ResponseText
    fMsg := JSON_Get(whr.ResponseText, "candidates[0].finishMessage")
    ;finishReason := JSON_Get(whr.ResponseText, "candidates[0].finishReason")

    if (fMsg != "") {
        ModelLogMsg("API MESSAGE: " . fMsg . "`n")
        LogMessage("API MESSAGE: " . fMsg . "`n") ; Log to debug.log
    }

    if RegExMatch(responseText, 's)"(data|bytesBase64Encoded)":\s*"([^"]+)"', &imgMatch) {
        binData := Base64ToBin(imgMatch[2])
        finalExt := (InStr(responseText, "image/png")) ? "png" : "jpg"
        outPath := OutputDir "\" nameNoExt "_" A_Now "." finalExt

        SaveBinaryImage(binData, outPath)

        ; Log the successful save location
        ModelLogMsg("Saved: " . outPath)
        if (batchIdx > 0)
            LV_Tasks.Modify(batchIdx, "", , , , , "Success")
    } else {
        idxLog := "Task " . batchIdx
        if RegExMatch(responseText, 'i)"message":\s*"([^"]+)"', &m) {
            ModelLogMsg("[ERROR]: " . idxLog . " - " . m[1])
        } else if RegExMatch(responseText, 'i)"finishReason":\s*"([^"]+)"', &m) {
            ModelLogMsg("[SAFETY]: " . idxLog . " - Reason: " . m[1])
        } else {
            ModelLogMsg("[FAILED]: " . idxLog . " - No image data returned.")
        }
        if (batchIdx > 0)
            LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
    }
}
    } catch Error as e {
        if (DEBUG)
            LogMessage("CRITICAL SCRIPT ERROR: " . e.Message . "`n")
        if (batchIdx > 0)
            LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
        ModelLogMsg("ERROR: " . e.Message)
    }
}

SaveBinaryImage(binBuffer, path) {
    try {
        if FileExist(path)
            FileDelete(path)

        fileObj := FileOpen(path, "w", "cp0") ; Open for writing in raw mode
        fileObj.RawWrite(binBuffer)            ; Write the raw buffer directly
        fileObj.Close()
    } catch Error as e {
        if (DEBUG)
            FileAppend("`n[" . FormatTime() . "] SAVE ERROR: " . e.Message, "debug.log")
        throw e
    }
}


; Helper to convert image file to Base64 string
FileToBase64(FilePath) {
    if !FileExist(FilePath)
        return ""

    FileObj := FileOpen(FilePath, "r")
    FileObj.RawRead(BinData := Buffer(FileObj.Length))
    FileObj.Close()

    ; Use Windows Crypt32 to encode
    DllCall("crypt32\CryptBinaryToString", "Ptr", BinData, "UInt", BinData.Size, "UInt", 0x40000001, "Ptr", 0, "UInt*", &Size := 0)
    VarSetStrCapacity(&Base64, Size)
    DllCall("crypt32\CryptBinaryToString", "Ptr", BinData, "UInt", BinData.Size, "UInt", 0x40000001, "Str", Base64, "UInt*", &Size)
    return StrReplace(StrReplace(Base64, "`r"), "`n")
}

CreateJsonPayload(taskObj, taskImagePath) {
    global encourageGen, encourageEdt, DEBUG

    isImagen := InStr(taskObj.Agent, "Imagen")

    if (isImagen) {
        promptText := "USER DIRECTIVE: " . taskObj.Prompt . ". Aspect Ratio: " . taskObj.Ratio

        payload := '{'
            . '"instances": [{"prompt": "' . StrReplace(StrReplace(promptText, '"', '\"'), "`n", " ") . '"}], '
            . '"parameters": {'
                . '"sampleCount": 1, '
                . '"aspectRatio": "' . taskObj.Ratio . '"'
                . (taskObj.Size != "1K" ? ', "sampleImageSize": "' . taskObj.Size . '"' : "")
            . '}'
        . '}'
        return payload
    }

    enc := ""
    fullPrompt := ""
    cleanPrompt := ""
    cleanEncourage := ""
    icfg := ""
    payload := ""

    ; Select encouragement based on whether we are generating from scratch or editing
    enc := (taskImagePath == "<GENERATE>") ? encourageGen : encourageEdt

    if (DEBUG)
        LogMessage("Payload creation for " . taskImagePath . " using " . ((taskImagePath == "<GENERATE>") ? "encourageGen" : "encourageEdt"))

    ; Merge instructions into the text prompt since Gemini doesn't support them in config
    fullPrompt := "USER DIRECTIVE: " . taskObj.Prompt
                . ". Aspect Ratio: " . taskObj.Ratio
                . ". Avoid: " . taskObj.NegativePrompt

    ; Sanitize prompt for JSON
    cleanPrompt := StrReplace(fullPrompt, '"', '\"')
    cleanPrompt := StrReplace(cleanPrompt, "`r", "")
    cleanPrompt := StrReplace(cleanPrompt, "`n", " ")

    cleanEncourage := StrReplace(enc, '"', '\"')
    cleanEncourage := StrReplace(cleanEncourage, "`r", "")
    cleanEncourage := StrReplace(cleanEncourage, "`n", " ")

    ;mimeStr := (taskObj.Format == "JPG") ? "image/jpeg" : "image/png"
    ; '"mime_type": "' . mimeStr . '"'
    ; '"mimeType": "' . mimeStr . '"'
    
    icfg := ''
     
    if (taskObj.Ratio != "Default")
        icfg .= ', "aspect_ratio": "' . taskObj.Ratio . '"'
        
    if (taskObj.Size != "1K")
        icfg .= ', "image_size": "' . taskObj.Size . '"'
        
    icfg := LTrim(icfg, ", ") ; trim the first ", " on the list.
    
    ;mime := (task.Format == "JPG") ? "image/jpeg" : "image/png"
    mime := (taskObj.Format = "PNG") ? "image/png" : "image/jpeg"

    if (taskImagePath == "<GENERATE>") {
         payload := '{'
             . '"contents": [{"parts": [{"text": "' . cleanPrompt . '"}]}], '
             . '"system_instruction": {"parts": [{"text": "' . cleanEncourage . '"}]}, '
             . '"safety_settings": ['
                 . '{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"}, '
                 . '{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"}, '
                 . '{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"}, '
                 . '{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"}'
             . '], '
             . '"generation_config": {'
                 . '"candidate_count": 1, '
                 . '"response_modalities": ["IMAGE"], '
                 . '"image_config": {' . icfg . '}, '
             . '}'
         . '}'
         return payload
    }

    imageParts := ""
    paths := StrSplit(taskImagePath, "|")
    for path in paths {
        if (path == "")
            continue
        if (FileExist(path)) {
         b64 := FileToBase64(path)
         imageParts .= ', {"inline_data": {"mime_type": "' . mime . '", "data": "' . b64 . '"}}'
        } else {
         ModelLogMsg("[Error] File not found:" . path)
         return "err"
        }
    }
    ModelLogMsg("[Okay] All files found:") ;xxx
    payload := '{'
        . '"contents": [{"parts": ['
            . '{"text": "' . cleanPrompt . '"}'
            . imageParts
        . ']}], '
        . '"system_instruction": {"parts": [{"text": "' . cleanEncourage . '"}]}, '
        . '"safety_settings": ['
            . '{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"}'
        . '], '
        . '"generation_config": {'
            . '"candidate_count": 1, '
            . '"response_modalities": ["IMAGE"], '
            . '"image_config": {' . icfg . '}'
        . '}'
    . '}'

    return payload
}

AsyncUploadBatchFile(FilePath, selectedModel) {
    global useCurl, API_KEY, CurlTimers
    if !FileExist(FilePath) {
        BatchError("Batch file not found: " . FilePath)
        return
    }

    fileData := FileRead(FilePath, "RAW")
    boundary := "-------AHKBoundary" . A_TickCount

    ; 1. Construct the multipart parts
    metadata := '{"file": {"display_name": "batch_job_' . A_Now . '"}}'

    bodyStart := "--" . boundary . "`r`n"
              . "Content-Type: application/json; charset=UTF-8`r`n`r`n"
              . metadata . "`r`n"
              . "--" . boundary . "`r`n"
              . "Content-Type: application/json`r`n`r`n"

    bodyEnd := "`r`n--" . boundary . "--`r`n"

    ; 2. Create the combined binary package
    size := (StrPut(bodyStart, "UTF-8") - 1) + fileData.Size + (StrPut(bodyEnd, "UTF-8") - 1)
    combinedBody := Buffer(size)

    offset := 0
    offset += StrPut(bodyStart, combinedBody, "UTF-8") - 1
    DllCall("RtlMoveMemory", "Ptr", combinedBody.Ptr + offset, "Ptr", fileData.Ptr, "Ptr", fileData.Size)

    offset += fileData.Size
    StrPut(bodyEnd, combinedBody.Ptr + offset, "UTF-8")

    if (useCurl) {
        tempBodyFile := A_ScriptDir . "\gemini_upload_" . A_TickCount . ".bin"
        resFile := A_ScriptDir . "\gemini_upload_res_" . A_TickCount . ".json"
        FileOpen(tempBodyFile, "w", "cp0").RawWrite(combinedBody)
        url := "https://generativelanguage.googleapis.com/upload/v1beta/files?key=" . API_KEY
        curlCmd := 'curl -s -X POST "' . url . '" -H "X-Goog-Upload-Protocol: multipart" -H "Content-Type: multipart/related; boundary=' . boundary . '" --data-binary "@' . tempBodyFile . '" -o "' . resFile . '"'

        Run(curlCmd, , "Hide", &pid)
        cb := ProcessBatchUploadCurl.Bind(pid, resFile, tempBodyFile, selectedModel)
        CurlTimers[pid] := cb
        SetTimer(cb, 100)
    } else {
        ; Convert Buffer to a Safe COM Stream
        pStream := DllCall("shlwapi\SHCreateMemStream", "Ptr", combinedBody.Ptr, "UInt", combinedBody.Size, "Ptr")
        IStream := ComValue(13, pStream) ; 13 = VT_UNKNOWN (IUnknown/IStream)

        whr := ComObject("WinHttp.WinHttpRequest.5.1")
        whr.SetTimeouts(30000, 60000, 600000, 600000)
        url := "https://generativelanguage.googleapis.com/upload/v1beta/files?key=" . API_KEY

        whr.Open("POST", url, true) ; Async
        whr.SetRequestHeader("X-Goog-Upload-Protocol", "multipart")
        whr.SetRequestHeader("Content-Type", "multipart/related; boundary=" . boundary)

        global ActiveStreams
        ActiveStreams[whr] := IStream ; Keep stream alive

        whr.Send(IStream)
        SetTimer(CheckWinHttpUpload.Bind(whr, selectedModel), 100)
    }
}

ProcessBatchUploadCurl(pid, resFile, tempBodyFile, selectedModel) {
    if ProcessExist(pid)
        return

    global CurlTimers
    if CurlTimers.Has(pid) {
        SetTimer(CurlTimers[pid], 0)
        CurlTimers.Delete(pid)
    }

    resText := ""
    if FileExist(resFile) {
        resText := FileRead(resFile)
        FileDelete(resFile)
    }
    if FileExist(tempBodyFile) {
        FileDelete(tempBodyFile)
    }

    if RegExMatch(resText, '"uri":\s*"([^"]+)"', &match) {
        LogMessage("BATCH UPLOAD SUCCESS: URI is " . match[1])
        AsyncSubmitBatchJob(match[1], selectedModel)
    } else {
        BatchError("Curl Upload Failed: " . resText)
    }
}

CheckWinHttpUpload(whr, selectedModel) {
    if (whr.ReadyState != 4)
        return

    SetTimer(, 0)
    global ActiveStreams
    if ActiveStreams.Has(whr)
        ActiveStreams.Delete(whr)

    if (whr.Status == 200) {
        if RegExMatch(whr.ResponseText, '"uri":\s*"([^"]+)"', &match) {
            LogMessage("BATCH UPLOAD SUCCESS: URI is " . match[1])
            AsyncSubmitBatchJob(match[1], selectedModel)
        } else {
            BatchError("Could not find URI in response: " . whr.ResponseText)
        }
    } else {
        BatchError("Multipart upload failed: " . whr.ResponseText)
    }
}


UpdateMonitorProgress() {
    global CurrentMonitorIndex, NextCheckTime, CheckInterval, batBar, LastFPress
    remaining := NextCheckTime - A_TickCount

    if (remaining <= 0) {
        ; 1. Find all currently active jobs
        activeJobs := []
        Loop batView.GetCount() {
            status := batView.GetText(A_Index, 2)
            if (RegExMatch(status, "i)^(BATCH_STATE_)?(Submitted|Checking|Processing|ACTIVE|RUNNING|PENDING|UNKNOWN)")) {
                activeJobs.Push({id: batView.GetText(A_Index, 1), row: A_Index})
            }
        }

        if (activeJobs.Length == 0) {
            SetTimer(UpdateMonitorProgress, 0)
            try batBar.Value := 0
            ModelLogMsg("Batch monitor: No active jobs. Stopping.")
            LastFPress := 0
            return
        }

        ; 2. Cycle through them
        if (CurrentMonitorIndex > activeJobs.Length)
            CurrentMonitorIndex := 1

        target := activeJobs[CurrentMonitorIndex]
        AsyncCheckBatchStatus(target.id, target.row)

        ; 3. Prepare for next check
        CurrentMonitorIndex += 1
        NextCheckTime := A_TickCount + CheckInterval
        ModelLogMsg("Batch monitor: Checked " . target.id . ". Next check in " . CheckInterval//1000 . "s")
        LastFPress := 0
          
        try batBar.Value := 0
    } else {
        try batBar.Value := Round((1 - (remaining / CheckInterval)) * 100)
    }
}

AsyncCheckBatchStatus(jobID, targetRow) {
    global useCurl, API_KEY, CurlTimers
    url := "https://generativelanguage.googleapis.com/v1beta/" . jobID . "?key=" . API_KEY

    if (useCurl) {
        resFile := A_ScriptDir . "\gemini_status_" . A_TickCount . "_" . targetRow . ".json"
        curlCmd := 'curl -s -L "' . url . '" -o "' . resFile . '"'
        LogMessage("Async status check: " . curlCmd)
        Run(curlCmd, , "Hide", &pid)

        cb := ProcessBatchStatus.Bind(pid, resFile, jobID, targetRow)
        CurlTimers[pid] := cb
        SetTimer(cb, 100)
    } else {
        ; Use a one-shot timer to make WinHttp also "async" from the monitor's perspective
        SetTimer(() => SyncCheckBatchStatus(url, jobID, targetRow), -10)
    }
}

SyncCheckBatchStatus(url, jobID, targetRow) {
    whr := ComObject("WinHttp.WinHttpRequest.5.1")
    try {
        whr.Open("GET", url, false)
        whr.Send()
        if (whr.Status == 200)
            HandleBatchStatus(whr.ResponseText, jobID, targetRow)
        else
            ModelLogMsg("[Error] WinHttp status " . whr.Status . " for " . jobID)
    } catch Error as e {
        ModelLogMsg("[Error] WinHttp status check failed: " . e.Message)
    }
}

ProcessBatchStatus(pid, resFile, jobID, targetRow) {
    if !ProcessExist(pid) {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        if FileExist(resFile) {
            responseText := FileRead(resFile)
            FileDelete(resFile)
            HandleBatchStatus(responseText, jobID, targetRow)
        }
    }
}

HandleBatchStatus(responseText, jobID, targetRow) {
    LogMessage("HandleBatchStatus for " . jobID . ": " . responseText . "...")
    state := JSON_Get(responseText, "state")
    ModelLogMsg(jobID . " = " . state)
    if (state == "") {
        if InStr(responseText, '"error"')
            state := "ERROR"
        else
            state := "UNKNOWN"

        LogMessage("Batch Job " . jobID . " returned no state. Full Response: " . responseText)
    }

    batView.Modify(targetRow, "", , state)
    batView.ModifyCol()
    LogMessage("Job " . jobID . " state: " . state)

    if (state == "SUCCEEDED" || state == "BATCH_STATE_SUCCEEDED") {
        outputUri := JSON_Get(responseText, "responsesFile")
        if (outputUri == "") {
             if RegExMatch(responseText, '"responsesFile":\s*"([^"]+)"', &m)
                 outputUri := m[1]
        }

        if (outputUri != "") {
            ModelLogMsg("Job " . jobID . " SUCCEEDED. Starting download from " . outputUri)
            LogMessage("Job " . jobID . " SUCCEEDED. Starting download from " . outputUri)
            AsyncDownloadBatch(outputUri, targetRow)
        } else {
            ModelLogMsg("[Warning] Job " . jobID . " succeeded but no responsesFile found.")
            LogMessage("[Warning] Job " . jobID . " succeeded but no responsesFile found in: " . responseText)
        }
    }
}

AsyncDownloadBatch(outputUri, targetRow) {
    global useCurl, API_KEY, CurlTimers
    finalUrl := "https://generativelanguage.googleapis.com/v1beta/" . outputUri . ":download?alt=media&key=" . API_KEY
    LogMessage("AsyncDownloadBatch URL: " . finalUrl)

    if (useCurl) {
        resFile := A_ScriptDir . "\gemini_batch_res_" . A_TickCount . "_" . targetRow . ".jsonl"
        curlCmd := 'curl -s -L "' . finalUrl . '" -o "' . resFile . '"'
        LogMessage("Async download: " . curlCmd)
        Run(curlCmd, , "Hide", &pid)

        cb := ProcessBatchDownload.Bind(pid, resFile, targetRow)
        CurlTimers[pid] := cb
        SetTimer(cb, 200)
    } else {
        SetTimer(() => SyncDownloadBatch(finalUrl, targetRow), -10)
    }
}

SyncDownloadBatch(finalUrl, targetRow) {
    whr := ComObject("WinHttp.WinHttpRequest.5.1")
    whr.SetTimeouts(30000, 60000, 600000, 600000)
    try {
        whr.Open("GET", finalUrl, false)
        whr.Send()
        if (whr.Status == 200)
            HandleBatchDownload(whr.ResponseText, targetRow)
    } catch Error as e {
        ModelLogMsg("[Error] WinHttp download failed: " . e.Message)
    }
}

ProcessBatchDownload(pid, resFile, targetRow) {
    if !ProcessExist(pid) {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        if FileExist(resFile) {
            responseText := FileRead(resFile)
            FileDelete(resFile)
            HandleBatchDownload(responseText, targetRow)
        }
    }
}

HandleBatchDownload(rawResponse, targetRow) {
    global OutputDir, batView
    jobID := batView.GetText(targetRow, 1)
    if (rawResponse == "") {
        ModelLogMsg("Error: Download response is empty.")
        LogMessage("HandleBatchDownload: rawResponse is EMPTY.")
        return
    }

    ModelLogMsg("Processing download (" . StrLen(rawResponse) . " bytes)...")
    LogMessage("HandleBatchDownload for " . jobID . ": Starting processing. Response length: " . StrLen(rawResponse))
    batView.Modify(targetRow, "", , "Success", , "100%")
    batView.ModifyCol()

    count := 0
    Loop Parse, rawResponse, "`n", "`r" {
        line := Trim(A_LoopField)
        if (line == "")
            continue

        LogMessage("Line " . A_Index . ": " . line . "...")

        fn := ""
        if RegExMatch(line, '"custom_id":\s*"([^"]+)"', &m)
            fn := m[1]

        LogMessage("Line " . A_Index . " ID: " . fn)

        ; Use a while loop to find ALL base64 data blocks in the line that look like images
        pos := 1
        foundInLine := 0
        while (pos := RegExMatch(line, 'i)"(data|processed_image_data)":\s*"([^"]{1000,})"', &m, pos)) {
            b64 := m[2]
            LogMessage("Found potential image data (" . m[1] . ") for " . fn . " at pos " . pos . ". Length: " . StrLen(b64))

            SplitPath(fn, &justFileName)
            try {
                binData := Base64ToBin(b64)
                outPath := OutputDir . "\Batch_" . A_TickCount . "_" . count . "_" . justFileName
                if !RegExMatch(outPath, "i)\.(jpg|png)$")
                    outPath .= ".jpg"

                SaveBinaryImage(binData, outPath)
                LogMessage("Saved: " . outPath)
                count++
                foundInLine++
            } catch Error as e {
                LogMessage("Error saving image: " . e.Message)
            }
            pos += m.Len
        }

        if (foundInLine == 0) {
             LogMessage("No images found for ID: " . fn)
             if (InStr(line, '"error"'))
                 LogMessage("Line " . A_Index . " error: " . line)
        }
    }

    if (count == 0) {
        ModelLogMsg("Warning: No images found in response for " . jobID . ". Dumping to failed_batch_response.json")
        LogMessage("HandleBatchDownload for " . jobID . ": NO IMAGES FOUND. Dumping to failed_batch_response.json")
        try {
            FileOpen(A_ScriptDir . "\failed_batch_response.json", "w", "UTF-8").Write(rawResponse)
        }
    } else {
        ModelLogMsg("Batch complete. Saved " . count . " images.")
    }
    CleanupJobsFile()
}
JSON_Get(jsonStr, key) {
    if RegExMatch(jsonStr, '"' . key . '":\s*"([^"]+)"', &match)
        return match[1]
    return ""
}



; Helper to convert the API's text response back to an image file
Base64ToBin(Base64Str) {
    ; Calculate the required buffer size
    DllCall("crypt32\CryptStringToBinary", "Str", Base64Str, "UInt", 0, "UInt", 0x1, "Ptr", 0, "UInt*", &Size := 0, "Ptr", 0, "Ptr", 0)
    BinData := Buffer(Size)
    ; Convert Base64 string to raw binary data
    DllCall("crypt32\CryptStringToBinary", "Str", Base64Str, "UInt", 0, "UInt", 0x1, "Ptr", BinData, "UInt*", &Size, "Ptr", 0, "Ptr", 0)
    return BinData
}


TestAPIConnection(*) {
    global useCurl, API_KEY
    ModelLogMsg("Fetching models...")
    Prog_Bar.Value := 10

    try {
        url := "https://generativelanguage.googleapis.com/v1beta/models?key=" . API_KEY
        responseText := ""
        status := 0
        if (useCurl) {
            resFile := A_ScriptDir . "\gemini_models_" . A_TickCount . ".json"
            curlCmd := 'curl -s "' . url . '" -o "' . resFile . '"'
            Run(curlCmd, , "Hide", &pid)
        while ProcessExist(pid)
            Sleep(50)
            if FileExist(resFile) {
                responseText := FileRead(resFile)
                FileDelete(resFile)
                status := 200
            }
        } else {
            whr := ComObject("WinHttp.WinHttpRequest.5.1")
            whr.Open("GET", url, false)
            whr.Send()
            status := whr.Status
            responseText := whr.ResponseText
        }

        if (status == 200) {
            Prog_Bar.Value := 100
            modelList := ""
            pos := 1
            while (pos := RegExMatch(responseText, "`"name`":\s*`"models/([^`"]+)`"", &match, pos + 1)) {
                modelList .= match[1] . "`r`n"
            }

            ModelLogMsg(modelList)

            LogMessage("--- SUPPORTED MODELS ---`n" . modelList . "`n")

        } else {
            ; Handle errors using your existing catch logic
            ModelLogMsg("Status " . whr.Status . ": " . whr.ResponseText)
        }
    } catch Error as e {
        Prog_Bar.Value := 0 ;
        ModelLogMsg("Connection Failed:" . e.Message)
    }
}

AutoSaveTask(*) {
    selectedTaskRow := LV_Tasks.GetNext(0, "Focused")
    if (selectedTaskRow == 0)
        return

    localIdx := Integer(LV_Tasks.GetText(selectedTaskRow, 9))
    
    ; Parse Tier string [cite: 298]
    RegExMatch(tier.Text, "(.*)\s(\d+K)", &match)
    agentName := (match) ? match[1] : "Unknown"
    agentSize := (match) ? match[2] : "1K"

    ; Update the task object in the Array
    if localIdx > 0 && localIdx <= ImageTaskMap.Length {
        task := ImageTaskMap[localIdx]

        ; Update ratio list if Tier changes
        if (task.Agent != agentName) {
            if InStr(agentName, "Nano 2") {
                ratio.Delete()
                ratio.Add(ratioListExt)
            } else {
                ratio.Delete()
                ratio.Add(ratioList)
            }
            ; Attempt to maintain previous ratio if it still exists
            try ratio.Text := task.Ratio
            if (ratio.Text == "")
                ratio.Value := 1
        }

        task.Prompt := ed.Value
        task.NegativePrompt := neg.Value
        task.Agent := agentName
        task.Size := agentSize
        task.Ratio := ratio.Text
        task.Format := fmt.Text
        task.Cost := CalculateCost(agentName, agentSize)
        
        ; Refresh the UI components
        RefreshTaskTable()
        UpdateTotalDisplay()
        
        ; Select the row again to maintain focus
        LV_Tasks.Modify(selectedTaskRow, "Select Focus")
        popCost.Value := "$" . Format("{:.3f}", task.Cost)
    }
    ;UpdateButtonStates()
    ValidateButtons()
    UpdateRatioOutline()
}

ValidateButtons() {
    taskCount := LV_Tasks.GetCount()
    imgCount  := LV_Images.GetCount()

    Btn_Add.Enabled := (imgCount > 0)
    btn_Save.Enabled := (taskCount > 0)
    btn_Run.Enabled := AreAgentsUniformAndBatchable()
}


GetClosestRatio(w, h) {
    target := w / h
    ratios := ["1:8","1:4","9:16","2:3","3:4","4:5","1:1","5:4","4:3","3:2","16:9","21:9","4:1","8:1"]
    bestMatch := "1:1"
    minDiff := 999.0

    for str in ratios {
        parts := StrSplit(str, ":")
        ratioVal := parts[1] / parts[2]
        diff := Abs(target - ratioVal)

        if (diff < minDiff) {
            minDiff := diff
            bestMatch := str
        }
    }
    return bestMatch
}


ClearFinishedJobs(*) {
    ; Start from the bottom and go to 1 to prevent index shifting
    idx := batView.GetCount()
    while (idx > 0) {
        status := batView.GetText(idx, 2) ; Column 2 is "Status"

        ; Check for every possible "finished" string state
        if (status == "Success" || status == "Failed" || status == "SUCCEEDED" || status == "BATCH_STATE_SUCCEEDED") {
            batView.Delete(idx)
        }
        idx-- ; Manually move to the next item up
    }

    ; Sync the jobs.txt file so deleted items don't return on reload [cite: 52, 57]
    CleanupJobsFile()
}

LogMessage(msg) {
    global DEBUG, SizeLimit, logPath
    if (!DEBUG)
        return
    
    CurrentSize := FileGetSize(logPath)
    timestamp := FormatTime(, "yyyy-MM-dd-HH-mm-ss")
    if (CurrentSize > SizeLimit) {
        SplitPath(logPath, &OutFileName, &OutDir, &OutExt, &OutNameNoExt)
        NewPath := OutDir "\(" timestamp ")." OutExt
        try {
            FileMove(logPath,NewPath,1)
        } catch as e {
            ModelLogMsg("Failed to rename log: " e.Message . " " . e.Extra . " " . A_LastError . "`n" . logPath " > " NewPath)
        }
    }
    FileAppend("[" . timestamp . "] " . msg . "`n", logPath, "UTF-8")
}

#HotIf WinActive("Gemini 2026 Pro Editor")
^d:: {
    textOut := "--- CURRENT IMAGETASKMAP ---`n`n"
    
    for index, task in ImageTaskMap {
        textOut .= "  Task #" . index . ":`n"
        textOut .= "    - Agent: " . task.Agent . " (" . task.Size . ")`n"
        textOut .= "    - Status: " . task.Status . "`n"
        textOut .= "    - ID: " . task.ID . "`n"
        textOut .= "    - IDs Involved: " . task.IDs . "`n"
        textOut .= "    - Prompt: " . SubStr(task.Prompt, 1, 50) . (StrLen(task.Prompt) > 50 ? "..." : "") . "`n"
    }
    
    MsgBox(textOut)
}
^r:: Reload()
^f:: {
    global LastFPress, NextCheckTime
    if (LastFPress > 0) {
        ModelLogMsg("CTRL+F = skip")
        return
    } else {
     LastFPress := 1
     ModelLogMsg("CTRL+F = force")
     NextCheckTime:=0
     UpdateMonitorProgress()
     sleep 250
    }
}
#HotIf

#HotIf ControlGetFocus(MyGui) == LV_Tasks.Hwnd || ControlGetFocus(MyGui) == LV_Images.Hwnd
$Del:: {
    FocusedCtrl := MyGui.FocusedCtrl

    if (FocusedCtrl == LV_Images) {
        Row := LV_Images.GetNext(0, "Focused")
        if (Row) {
            imgID := LV_Images.GetText(Row, 1)

            ; Identify all tasks that involve this image ID and remove them
            i := ImageTaskMap.Length
            while i > 0 {
                if (IsIDInMergedID(imgID, ImageTaskMap[i].IDs)) {
                    ImageTaskMap.RemoveAt(i)
                }
                i--
            }

            LV_Images.Delete(Row)

            ; Update task counts for all remaining images
            Loop LV_Images.GetCount() {
                oid := LV_Images.GetText(A_Index, 1)
                taskCount := 0
                for t in ImageTaskMap {
                    if (IsIDInMergedID(oid, t.IDs)) {
                        taskCount++
                    }
                }
                LV_Images.Modify(A_Index, "", , , taskCount)
            }

            RefreshTaskTable()
            UpdateTotalDisplay()

            if (LV_Images.GetCount() > 0) {
                NewRow := (Row > LV_Images.GetCount()) ? LV_Images.GetCount() : Row
                LV_Images.Modify(NewRow, "Select Focus")
                ; ItemFocus event will handle ImageListClick
            }
        }
    }
    else if (FocusedCtrl == LV_Tasks) {
        Row := LV_Tasks.GetNext(0, "Focused")
        if (Row) {
            taskIdx := Integer(LV_Tasks.GetText(Row, 9))

            if (taskIdx > 0 && taskIdx <= ImageTaskMap.Length) {
                targetIDs := ImageTaskMap[taskIdx].IDs
                ImageTaskMap.RemoveAt(taskIdx)

                ; Update task count in LV_Images for ALL involved images
                ids := StrSplit(targetIDs, ",")
                for id in ids {
                    Loop LV_Images.GetCount() {
                        if (LV_Images.GetText(A_Index, 1) == id) {
                            taskCount := 0
                            for t in ImageTaskMap {
                                if (IsIDInMergedID(id, t.IDs)) {
                                    taskCount++
                                }
                            }
                            LV_Images.Modify(A_Index, "", , , taskCount)
                            break
                        }
                    }
                }
            }

            RefreshTaskTable()
            UpdateTotalDisplay()
            UpdateButtonStates()
        }
    }
}
#HotIf



IsIDInMergedID(id, mID) {
    sid := String(id)
    smID := String(mID)
    if (smID == sid)
        return true
    for _, p in StrSplit(smID, ",") {
        if (p == sid)
            return true
    }
    return false
}

ModelLogMsg(txt) {
    global ModelLog
    try {
        timestamp := FormatTime(, "HH:mm:ss")
        ModelLog.Value .= "`n[" . timestamp . "] " . txt
        SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A") ; WM_VSCROLL = 0x0115, SB_BOTTOM = 7
    }
}

; --- Async Curl Helpers ---
CheckCurlProgress(pid, responseFile, payloadFile, batchIdx, nameNoExt) {
    if !ProcessExist(pid) {
        ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt)
        return
    }

    if FileExist(responseFile) {
        try {
            fileContent := FileRead(responseFile)
            ; "Drop the stream": Wait until the image data field is closed by a quote
            ; Use InStr for performance on large stream files
            p1 := InStr(fileContent, '"data":')
            if (!p1)
                p1 := InStr(fileContent, '"bytesBase64Encoded":')

            if (p1) {
                pStart := InStr(fileContent, ":", , p1)
                if (pStart && p2 := InStr(fileContent, '"', , pStart)) {
                    if (p3 := InStr(fileContent, '"', , p2 + 1)) {
                        ProcessClose(pid)
                        ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt)
                        return
                    }
                }
            }
        }
    }
}

ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt) {
    global DEBUG
    try {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        responseText := ""
        if FileExist(responseFile) {
            responseText := FileRead(responseFile)
            FileDelete(responseFile)
        }
        checkx := InspectApiResponse(responseText,batchIdx
        )
        
        ;idxLog := (batchIdx < 0) ? "Upscale " . Abs(batchIdx) : "Task " . batchIdx
        idxLog := "Task " . batchIdx
        LogMessage(idxLog . " Response: " . responseText)

        ; Try to find curl error log if response is empty
        if (responseText == "") {
            Loop Files, A_ScriptDir . "\gemini_curl_err_*.log" {
                if InStr(A_LoopFileName, "_" . batchIdx . ".log") {
                    errText := FileRead(A_LoopFileFullPath)
                    LogMessage(idxLog . " Curl Error: " . errText)
                    FileDelete(A_LoopFileFullPath)
                    break
                }
            }
        }

        if FileExist(payloadFile)
           FileDelete(payloadFile)

        global PendingTasks -= 1

        if (responseText != "") {
            ; Use InStr/SubStr for robust extraction from potentially huge JSON strings
            p1 := InStr(responseText, '"data":')
            if (!p1)
                p1 := InStr(responseText, '"bytesBase64Encoded":')

            if (p1) {
                pStart := (InStr(SubStr(responseText, p1), ":") + p1)
                p2 := InStr(responseText, '"', , pStart)
                p3 := InStr(responseText, '"', , p2 + 1)
                if (p2 && p3) {
                    base64Data := SubStr(responseText, p2 + 1, p3 - p2 - 1)

                    ; Detect extension
                    mime := "image/png"
                    if RegExMatch(responseText, '"mimeType":\s*"([^"]+)"', &mimeMatch)
                        mime := mimeMatch[1]
                    ext := (InStr(mime, "jpeg") || InStr(mime, "jpg")) ? "jpg" : "png"

                    outPath := OutputDir . "\" . nameNoExt . "_" . A_Now . "." . ext

                    try {
                        size := 0
                        if DllCall("crypt32\CryptStringToBinary", "Str", base64Data, "UInt", 0, "UInt", 1, "Ptr", 0, "UInt*", &size, "Ptr", 0, "Ptr", 0) {
                            buf := Buffer(size)
                            if DllCall("crypt32\CryptStringToBinary", "Str", base64Data, "UInt", 0, "UInt", 1, "Ptr", buf, "UInt*", &size, "Ptr", 0, "Ptr", 0) {
                                FileOpen(outPath, "w").RawWrite(buf)
                                ModelLogMsg("Image saved: " . outPath)
                                if (batchIdx > 0)
                                    LV_Tasks.Modify(batchIdx, "", , , , , "Success")
                            }
                        }
                    } catch as e {
                        ModelLogMsg("Error decoding image: " . e.Message)
                        if (batchIdx > 0)
                            LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
                    }
                } else {
                    ModelLogMsg("Could not find complete image data in curl response.")
                    if (batchIdx > 0)
                        LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
                }
            } else {
                if (RegExMatch(responseText, 'i)"finishReason":\s*"([^"]+)"', &m)) {
                    ModelLogMsg("Curl task " . batchIdx . " failed. Reason: " . m[1])
                } else {
                    ModelLogMsg("Curl task " . batchIdx . " returned no image data. See debug.log.")
                }
                if (batchIdx > 0)
                    LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
            }
        } else {
            ModelLogMsg("Curl task " . batchIdx . " finished with no output.")
            if (batchIdx > 0)
                LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
        }
    } catch as e {
        ModelLogMsg("Critical error in ProcessCurlResult: " . e.Message)
    }

    CheckQueueCompletion()
}

CheckQueueCompletion() {
    global PendingTasks
    if (PendingTasks <= 0) {
        PendingTasks := 0
        ToggleUI(true)
        ModelLogMsg("All tasks completed.")
    }
}

CleanupJobsFile() {
    jobFile := A_ScriptDir . "\jobs.txt"
    outString := ""

    Loop batView.GetCount() {
        jobID  := batView.GetText(A_Index, 1)
        status := batView.GetText(A_Index, 2)

        isFinished := (status == "Success" || status == "Failed" || status == "SUCCEEDED" || status == "BATCH_STATE_SUCCEEDED" || status == "FAILED" || status == "CANCELLED")

        if (!isFinished) {
            outString .= jobID . "`n"
        }
    }

    try {
        if FileExist(jobFile)
            FileDelete(jobFile)

        if (outString != "")
            FileAppend(outString, jobFile)

        ModelLogMsg("jobs.txt updated (cleaned completed jobs).")
    } catch Error as e {
        ModelLogMsg("[Error] Failed to update jobs.txt: " . e.Message)
    }
}

InspectApiResponse(jsonString,batchIdx) {
    if (InStr(jsonString, '"error"')) {
        code := JSON_Get(jsonString, "error.code")
        msg := JSON_Get(jsonString, "error.message")
        ModelLogMsg(batchIdx . " ERROR:" . code . " - " . msg)
    }

    if (InStr(jsonString, '"finishReason":"IMAGE_SAFETY"')) {
       ModelLogMsg(batchIdx . " IMAGE_SAFETY: Nano generated an image that tripped a safety policy! Try another image or rephrasing the prompt.")
    }
    return
}

AreAgentsUniformAndBatchable() {
    taskCount := LV_Tasks.GetCount()
    if (taskCount < 1)
        return false

    if (Btn_Run.Text == "RUN IMMEDIATE")
      return true
    
    firstAgent := ""
    Loop taskCount {
        currentAgent := LV_Tasks.GetText(A_Index, 2) ; Column 2 is the Agent name
        
        ; 1. Rule: Imagen models cannot run in batch
        if InStr(currentAgent, "Imagen")
            return false
            
        ; 2. Rule: All agents in the list must be identical
        if (A_Index == 1) {
            firstAgent := currentAgent
        } else if (currentAgent != firstAgent) {
            return false
        }
    }
    return true
}

UpdateRatioOutline() {
    global Pic_Preview, ratio, OutT, OutB, OutL, OutR

    if (!ratio.Enabled || ratio.Text == "Default" || ratio.Text == "") {
        OutT.Visible := false
        OutB.Visible := false
        OutL.Visible := false
        OutR.Visible := false
        return
    }

    try {
        parts := StrSplit(ratio.Text, ":")
        if (parts.Length != 2)
            return
        targetRatio := parts[1] / parts[2]
    } catch {
        return
    }

    Pic_Preview.GetPos(&px, &py, &pw, &ph)
    thick := 4
    
    if (pw<1 || ph<1){
      pw:=300
      ph:=200
      Pic_Preview.Value := "*w" pw*(A_ScreenDPI/96) " *h" ph*(A_ScreenDPI/96)
    }
    ;tooltip px " " py " " pw " " ph
    if (pw/ph > targetRatio) {
        bh := ph
        bw := bh * targetRatio
    } else {
        bw := pw
        bh := bw / targetRatio
    }

    bx := px + (pw - bw) / 2
    by := py + (ph - bh) / 2

    OutT.Move(bx, by, bw, thick)
    OutB.Move(bx, by + bh - thick, bw, thick)
    OutL.Move(bx, by, thick, bh)
    OutR.Move(bx + bw - thick, by, thick, bh)

    OutT.Visible := true
    OutB.Visible := true
    OutL.Visible := true
    OutR.Visible := true
}

UpdateRatioBars() {
 global OutT,OutB,OutL,OutR,outsz
 bgc:= ["Green","Red","Blue","Yellow"]
 bg:=3
 ;tooltip bgc[bg]
 ;OutT.Opt("+c" bgc[bg])
 ;OutB.Opt("+c" bgc[bg])
 ;OutL.Opt("+c" bgc[bg])
 ;OutR.Opt("+c" bgc[bg])
 outsz+=1
 if (outsz>100) {
  outsz:=0
 }
 OutT.Value:=outsz
 OutB.Value:=100-outsz
 OutL.Value:=outsz
 OutR.Value:=100-outsz
}
