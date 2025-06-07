import dayjs from "dayjs";
import duration from "dayjs/plugin/duration";
import * as message from "./message.js"
dayjs.extend(duration);
const controller = new AbortController();

// console.warn(`%cWARNING:\n%cNever paste any code here unless you know EXACTLY what you're doing.`,"color: red; font-weight: bold; font-size: 4em;","font-size: 2em;")

interface Dimension {
  id: string;
  title: string;
  encrypted: Boolean;
  visibility: boolean;
  fileName: string;
  text: string;
  downloadLimit: number;
  expirationDate: number;
  expirationDateISO: string;
}
let publicIndex = 0;
let itemsPerPage = 5;


let downloadLimit = document.getElementById(
  "downloadLimit",
) as HTMLInputElement;
let unlCheck = document.getElementById("unlimitedCheckbox") as HTMLInputElement;

// creates and returns a item(li element) containing some info 
// from a dimension
function createDimensionWindow(dim: Dimension): HTMLLIElement {
  const byteSize = str => new Blob([str]).size;

  let item = document.createElement("li");

  let a = document.createElement("a") as HTMLAnchorElement;
  a.setAttribute("class","title")
  a.setAttribute("href", "grab?id=" + dim.id);
  a.innerHTML = dim.title;
  
  let bytes = document.createElement("p") as HTMLParagraphElement;
  bytes.innerHTML = `Size: ${byteSize(dim.text)} bytes`

  let expirationDate = document.createElement("p") as HTMLParagraphElement;
  expirationDate.innerHTML = `Expiration Date: ${dim.expirationDateISO}`

  item.appendChild(a);
  item.appendChild(bytes);
  item.appendChild(expirationDate)

  return item;
}

// Remove old list from dom and adds dimensions(li element) to ordered list
// The start and end of the requested dimensions are 
// expressed as index and numDims
async function retrievePublicDimensions(index: number, numDims: number) {
  let params = {
    index: index.toString(),
    numDims: numDims.toString(),
  };
  let searchParams = new URLSearchParams(params);
  console.log(`api/public?${searchParams.toString()}`)
  let res = await fetch(`api/public?${searchParams.toString()}`, {
    method: "GET",
  });

  let dimensions = await res.json();
  let list = document.getElementById("list") as HTMLOListElement;

  // pruges any children in the list for the new list
  while (list.hasChildNodes()) {
    list.removeChild(list.children[0]);
  }

  for (let i in dimensions) {
    let dim = dimensions[i];
    if (dim.id == ""){continue}

    console.log(dim["title"])
    let item = createDimensionWindow(dim);

    list.appendChild(item);
  }
}

// Sends dimension in json format to the server and
// redirects the page to a view the created dimension
async function sendMessage() {

  let title = document.getElementById("title") as HTMLInputElement;
  let vis = document.getElementById("visibility") as HTMLInputElement;
  let expiry = document.getElementById("expiry") as HTMLInputElement;
  let password = document.getElementById("password") as HTMLInputElement;
  let content = document.getElementById("content") as HTMLInputElement;
  let fileInput = document.getElementById("fileInput") as HTMLInputElement;

  
  let msg = new message.Message(title,vis,expiry,password,content,fileInput,unlCheck,downloadLimit);
  // To encrypt, or not to encrypt; that is the question
  if (msg.password.value != "") {
    await msg.encryptClientData();
  } else {
    await msg.plainDimension();
  }

  let { rsaProtectedAES, combinedIVAndDimension } = await msg.encryptPayload();

  if (msg.file) {
    console.log("file name ----->", msg.file.name);
  }
  console.log(msg.dimension);

  // Server returns id and auth token for a file upload.
  // Id is the same for both the file and the dimension.
  // Token is empty if the json attribute known as fileName
  // within dimension is empty(no file given).
  let res = await msg.sendDimension(rsaProtectedAES, combinedIVAndDimension);
  const response = await res.json();
  console.log("Server response:", response);

  // If file exists it uploads one file. The token is used to autheticate.
  // This makes it harder to spam the tus sever. Its terrible at preventing
  // humans from using it as a datebase, but probably
  // effective at stopping bots(tus is a popular protocol).
  if (msg.file) {
    msg.uploadFile(response["id"], response["token"]);
  }

  // redirects the page to a view the created dimension
  msg.redirect(response["id"]);
}



retrievePublicDimensions(publicIndex, itemsPerPage);


unlCheck.addEventListener("change", () => {
  downloadLimit.disabled = unlCheck.checked;
});

document.getElementById("submit")!.addEventListener("click", sendMessage);
document.getElementById("refresh")!.addEventListener("click", ()=>{
  let svg = document.getElementById("svgRefresh");
  svg?.classList.add('spinning');
  svg?.addEventListener('animationend', () => {
    svg.classList.remove('spinning');
  }, { once: true });


  retrievePublicDimensions(publicIndex, itemsPerPage);
})

function recalculatePageNum(){
  let page = document.getElementById("page") as HTMLHeadingElement
  if(publicIndex === 0){
    page.innerHTML = "0"
    return
  }

  page.innerHTML = String(publicIndex/itemsPerPage)
}

document.getElementById("left")!.addEventListener("click",()=>{
  if (publicIndex - itemsPerPage < 0){
    console.warn("public item index cannot be less than 0")
    return
  }
  publicIndex -= itemsPerPage

  recalculatePageNum()
  retrievePublicDimensions(publicIndex, itemsPerPage);
})


document.getElementById("right")!.addEventListener("click",()=>{
  publicIndex += itemsPerPage

  recalculatePageNum()
  retrievePublicDimensions(publicIndex, itemsPerPage);
})
