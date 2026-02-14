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
From my testing it's usually like 10-20 mins. 4K files take longer to make than 1K, in pro or flash.<br>
<br>
<b>Images:</b><br>
<img src="main.png"><br>
<img src="batch.png"><br>



