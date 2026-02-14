You probably want the latest version of this script. It's probably the least buggy.<br>
You also need AHK AutoHotKey v2. https://www.autohotkey.com/download/<br>
Internet is also required, images get uploaded to google servers. You can start a batch and go offline, the app will check once it's online again.<br>
<br>
NOTE: I have all the safety settings disabled, to enable, use any of the other settings; BLOCK_NONE, BLOCK_ONLY_HIGH, BLOCK_MEDIUM_AND_ABOVE, BLOCK_LOW_AND_ABOVE<br>
The more images get blocked the farther down the list you go. Despite the settings, google will block images automatically if it sets off a filter.<br>
<code>
        . '"safetySettings": ['
            . '{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"}'
</code><br>
<br>
This AI image Editor/Generator was a collaboration between me and google gemini/jules.<br>
It all started when I was asking about the nano banana and nano banana pro API and it offered to write a bat file.
So I asked, "how about an AHK instead?", and here we are weeks later...
it works, it's ugly and probably buggy, it freezes when doing http.. but it's free. ;)<br>
<br>
Before you complain about something, visit https://paypal.me/ctg3d <br>
I can totally be encouraged to fix/upgrade things faster...<br>
<br>
<b>Getting started:</b><br>
 Get your API key from google, see AHK source for URL, drop your key in the code and run the app.<br>
 "log in to https://aistudio.google.com/ create new project, then create an API key."<br>
<br>
<b>Never Used AHK?<b><br>
if you install AHK program, you can run my AHK script just like a program.<br>
If you run it portable (extracted AHK from a zip file), you drag my AHK you downloaded and drop it on AutoHotkey64.exe<br>
You can also compile AHK scripts into their own *.exe file.<br>
<br>
<b>Usage:</b><br>
 Drag and drop images to edit OR click Generate to make an image from scratch.<br>
 Pick an image in the list or choose the GENERATE row.<br>
 Click the "add task" button<br>
 Fill out the popup info. The ratio defaults to the closest of the image. *1<br>
 Choose immediate or batch by radio select. *2<br>
 Click Run Immediate/Batch button, wait for the magic.<br>
 Check your img folder where you ran the script for new images.<br>
 Then repeat<br>
<br>
<b>Actions:</b><br>
 Use shift or ctrl to pick several images, then press "add task".<br>
 double click to browse/change the image on the image list.<br>
 double click to edit the task popup on the task list.<br>
 Delete key deletes the selected image/task.<br>
 CTRL+r = reload, great for clearing the list.<br>
 You can save and load your prompts to CSV files.<br>
 The test API key lists the AI models you can run on the key you provide. *3<br>
 <br>
<b>*1 prompt help **</b><br>
 If you have multiple images, say a hat and a person, you can say stuff like "put the hat on the person".<br>
 If you ae making an image from scratch, make sure to let the AI know what you want to see;<br>
  backgrounds, foreground, subjects, camera shot (full body, portrait, etc) and so on.<br>
  if you leave anything out, the AI will make something up.<br>
<br>
Keep in mind, the AI will still likely make things up anyway, it's a bit of a free spirit.<br>
<br>
<b>*2 immediate or batch? **</b><br>
Immediate does the whole upload and recieve image in one go, one at a time. You can mix and match between flash and pro.<br>
Batch lets you queue up images on your task list to run when the GPUs at google aren't busy, the list has to be all pro or all flash.<br>
Batch is cheaper and can take up to 24h, but typically, if the batch is small and the servers aren't busy, it's much less<br>
Batches are on a timer, the green bar. It automatically checks if the files are done as long as the app is open.<br>
From my testing it's usually like 10-20 mins. 4K files take longer to make than 1K, in pro or flash.<br>
Batch lookup numbers are also saved to disk, so you can close the app and come back later to download.<br>
<br>
*3 other APIs **<br>
 This script is only set up for the two it uses, but you can adapt it for more.<br>
 Here are some of mine that it listed, you'll have to figure out how they work on your own or ask google gemini.<br>
 I think some of these could be chat API's, some say audio, veo I assume is video...
<code>
gemini-2.5-flash
gemini-2.5-pro
gemini-2.0-flash
gemini-2.0-flash-001
gemini-2.0-flash-exp-image-generation
gemini-2.0-flash-lite-001
gemini-2.0-flash-lite
gemini-exp-1206
gemini-2.5-flash-preview-tts
gemini-2.5-pro-preview-tts
gemma-3-1b-it
gemma-3-4b-it
gemma-3-12b-it
gemma-3-27b-it
gemma-3n-e4b-it
gemma-3n-e2b-it
gemini-flash-latest
gemini-flash-lite-latest
gemini-pro-latest
gemini-2.5-flash-lite
gemini-2.5-flash-image
gemini-2.5-flash-preview-09-2025
gemini-2.5-flash-lite-preview-09-2025
gemini-3-pro-preview
gemini-3-flash-preview
gemini-3-pro-image-preview
nano-banana-pro-preview
gemini-robotics-er-1.5-preview
gemini-2.5-computer-use-preview-10-2025
deep-research-pro-preview-12-2025
gemini-embedding-001
aqa
imagen-4.0-generate-preview-06-06
imagen-4.0-ultra-generate-preview-06-06
imagen-4.0-generate-001
imagen-4.0-ultra-generate-001
imagen-4.0-fast-generate-001
veo-2.0-generate-001
veo-3.0-generate-001
veo-3.0-fast-generate-001
veo-3.1-generate-preview
veo-3.1-fast-generate-preview
gemini-2.5-flash-native-audio-latest
gemini-2.5-flash-native-audio-preview-09-2025
gemini-2.5-flash-native-audio-preview-12-2025
</code>

<b>Images:</b><br>
<img src="main.png"><br>
<img src="batch.png"><br>

