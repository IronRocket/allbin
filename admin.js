let youtube_scaled = false;
let pingState = false;
input.addEventListener("keypress", function(event) {
    if (event.key === "Enter") {
        document.getElementById("request").click()
    }
});

function sleep(ms) {
    return new Promise(r => setTimeout(r, ms))
}

function dark() {
    if (youtube_scaled === true) {
        anime({
            targets: "#scale",
            translateY: 0,
            duration: 1e3,
            easing: "easeInOutExpo"
        });
        youtube_scaled = false;
        document.getElementById("audio").setAttribute("width", "0");
        document.getElementById("audio").setAttribute("height", "0")
    } else {
        anime({
            targets: "#scale",
            translateY: -350,
            duration: 1500,
            easing: "spring(1, 80, 10, 0)"
        });
        youtube_scaled = true;
        document.getElementById("audio").setAttribute("width", "350");
        document.getElementById("audio").setAttribute("height", "350")
    }
}

function changePingState() {
    if (pingState) {
        pingState = false
    } else {
        pingState = true;
        ping()
    }
}
let serverPublickey;
let keyPair;
let publicKeyPem;
let aes;
let aesKey;
let decoder = new TextDecoder;
async function generateKeys() {
    keyPair = await window.crypto.subtle.generateKey({
        name: "RSA-OAEP",
        modulusLength: 4096,
        publicExponent: new Uint8Array([1, 0, 1]),
        hash: "SHA-256"
    }, true, ["encrypt", "decrypt"]);
    publicKeyPem = await window.crypto.subtle.exportKey("spki", keyPair.publicKey);
    aes = await crypto.subtle.generateKey({
        name: "AES-GCM",
        length: 256
    }, true, ["encrypt", "decrypt"]);
    aesKey = await crypto.subtle.exportKey("raw", aes)
}
async function decrypt(cipher) {
    let arr = Uint8Array.from(atob(cipher.slice(0, cipher.indexOf("iv:"))), c => c.charCodeAt(0));
    let ivt = Uint8Array.from(atob(cipher.slice(cipher.indexOf("iv:") + 3, cipher.length)), c => c.charCodeAt(0));
    let decrypted = await crypto.subtle.decrypt({
        name: "AES-GCM",
        iv: ivt
    }, aes, arr.buffer);
    return decoder.decode(decrypted)
}
async function ping() {
    let x = new XMLHttpRequest;
    x.onload = function() {
        delay = Date.now() - Number(JSON.parse(this.responseText));
        document.getElementById("ping").innerHTML = delay.toString() + "ms"
    };
    console.log("started pinging");
    document.getElementById("pingState").style = "color:green";
    while (pingState) {
        x.open("GET", "/ping", true);
        x.getResponseHeader("Content-type", "application/json; charset=UTF-8");
        x.send();
        await sleep(750)
    }
    document.getElementById("pingState").style = "color:red";
    console.log("stopped pinging")
}
async function basic() {
    document.getElementById("request").disabled = true;
    document.getElementById("server_status").style.color = "green";
    let x = new XMLHttpRequest;
    x.onload = async function() {
        document.getElementById("server_response").innerHTML = await decrypt(this.responseText);
        document.getElementById("server_status").style.color = "#192130"
    };
    let encrypted = await crypto.subtle.encrypt({
        name: "RSA-OAEP"
    }, serverPublickey, (new TextEncoder).encode(document.getElementById("input").value));
    x.open("POST", "/basic", true);
    x.getResponseHeader("Content-type", "application/json; charset=UTF-8");
    x.setRequestHeader("Authorization", "Basic " + btoa("admin" + ":" + "GYeTP2XK8uVUW44!i#QMyC9LdsJ44E#o52s*gJ$uC"));
    x.send(btoa(String.fromCharCode.apply(null, new Uint8Array(encrypted))) + "aesKey:" + btoa(String.fromCharCode.apply(null, new Uint8Array(aesKey))));
    document.getElementById("request").disabled = false
}

function publicKey() {
    let server = new XMLHttpRequest;
    server.onload = async function() {
        const raw_key = window.atob(this.responseText.substring(31, this.responseText.length - 29));
        const buffer = new ArrayBuffer(raw_key.length);
        const bytes = new Uint8Array(buffer);
        for (let i = 0, strLen = raw_key.length; i < strLen; i++) {
            bytes[i] = raw_key.charCodeAt(i)
        }
        serverPublickey = await crypto.subtle.importKey("spki", bytes.buffer, {
            name: "RSA-OAEP",
            hash: "SHA-256"
        }, true, ["encrypt"])
    };
    server.open("GET", "/public", true);
    server.getResponseHeader("Content-type", "text/plain; charset=UTF-8");
    server.send()
}
window.changePingState = changePingState;
publicKey();
generateKeys();
