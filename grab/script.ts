import * as ciphs from "./decryption.ts";

interface Dimension {
  id: string;
  title: string;
  encrypted: Boolean;
  visibility: boolean;
  fileName: string;
  text: string;
  reads: number;
  downloadLimit: number;
  ExpirationDate: number;
  ExpirationDateISO: string;
  createdAt: number;
}

let password: string;

async function decryptFile(dimension: Dimension) {
  let linkHref = "/files/" + dimension["FilePath"];
  if (dimension["encrypted"]) {
    const response = await fetch(linkHref);
    if (!response.ok) throw new Error("Failed to download file");
    const encryptedBuffer = await response.arrayBuffer();

    const decryptedBuffer = await ciphs.decryptWithPassword(
      password,
      new Uint8Array(encryptedBuffer),
    );

    const blob = new Blob([decryptedBuffer]);
    linkHref = URL.createObjectURL(blob);
  }
  const link = document.createElement("a");
  link.href = linkHref;
  link.download = dimension["fileName"];
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

async function DisplayInfo(dimension: Dimension) {
  let content = document.getElementById("data") as HTMLInputElement;
  let reads = document.getElementById("reads") as HTMLInputElement;
  let expirationDate = document.getElementById(
    "expirationDate",
  ) as HTMLInputElement;

  content.textContent = dimension["text"];
  reads.textContent = String(dimension["reads"]);
  expirationDate.textContent = dimension["expirationDateISO"];

  if (dimension["fileName"] != "") {
    let fileCard = document.getElementById("fileCard") as HTMLElement;
    fileCard.style = "display: block";
    let btn = document.getElementById("downloadBtn") as HTMLInputElement;
    btn.textContent = dimension["fileName"];
    btn.addEventListener("click", () => {
      decryptFile(dimension);
    });
  }

  console.log(dimension);
}

async function retrieveDimension(id: string): Promise<Dimension> {
  // Now you can use it for fetching data
  console.log("api grab: ", "/api/grab/" + encodeURIComponent(id));
  let res = await fetch("/api/grab/" + encodeURIComponent(id), {
    method: "GET",
  });

  const dimension = await res.json();
  console.log("raw response", dimension);
  return dimension;
}

function retrieveId(): string {
  const params = new URLSearchParams(window.location.search);
  const id = params.get("id");

  if (id) {
    return id;
  } else {
    console.error("Missing ID in query");
    return "";
  }
}

function waitForSubmit(id: string): Promise<Event> {
  const el = document.getElementById(id);
  if (!el) {
    return Promise.reject(new Error(`Element ${id} not found`));
  }


  return new Promise((resolve) => {
    el.addEventListener("click", resolve, { once: true });
  });
}

async function main() {
  let id = retrieveId();

  let dimension = await retrieveDimension(id);

  if (dimension["encrypted"]) {
    let passwordDiv = document.getElementById(
      "passwordModal",
    ) as HTMLInputElement;
    let passwordElm = document.getElementById("password") as HTMLInputElement;

    passwordDiv.style = "display: flex";
    console.log("waiting for input...");

    await waitForSubmit("submit");
    
    passwordDiv.style = "display: none";

    password = passwordElm.value;
    dimension["text"] = await ciphs.base64decryptWithPassword(
      password,
      dimension["text"],
    );

    if (dimension["fileName"] != ""){
      dimension["fileName"] = await ciphs.base64decryptWithPassword(
        password,
        dimension["fileName"],
      );
    }
  }
  DisplayInfo(dimension);
}

main();
