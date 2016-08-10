var start = new Date();

var i = 0;
//var response = $sendSync('Message from JS');
//var obj = JSON.parse(response);
function generateUUID () {
    return 'xxxxxxxxxxxx4xxxyxxxxxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

var returnArray = [];
while (i < 500){
    returnArray.push(
        {
            ProcessedEntry: {name:"asd",type:"asd",id:generateUUID()}
        }
    );
    i++
}
var j = 0;

var asd = JSON.stringify(returnArray);
var end = new Date();
var totalTime = end - start;
$send(JSON.stringify(totalTime))
