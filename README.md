You probably want the latest version of this script.<br>
You also need AHK AutoHotKey v2. https://www.autohotkey.com/download/<br>
<br>
NOTE: I have all the safety settings disabled, use any of the other ones to re-enable.
<code>
        . '"safetySettings": ['
            . '{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"}'
</code><br>
Settings are; BLOCK_NONE, BLOCK_ONLY_HIGH, BLOCK_MEDIUM_AND_ABOVE, BLOCK_LOW_AND_ABOVE, HARM_BLOCK_THRESHOLD_UNSPECIFIED<br>
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

<b>Usage:</b><br>
 Drag and drop images to edit OR click Generate to make an image from scratch.<br>
 Pick an image in the list or choose the GENERATE row.<br>
 Click the "add task" button<br>
 Fill out the popup info. The ratio defaults to the closest of the image. **<br>
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

<b>** prompt help **</b><br>
 If you have multiple images, say a hat and a person, you can say stuff like "put the hat on the person".<br>
 If you ae making an image from scratch, make sure to let the AI know what you want to see;<br>
  backgrounds, foreground, subjects, camera shot (full body, portrait, etc) and so on.<br>
  if you leave anything out, the AI will make something up.<br>
<br>
Keep in mind, the AI will still likey make things up any way, it's a bit of a free spirit.<br>




<img src="main.png"><img src="batch.png">


