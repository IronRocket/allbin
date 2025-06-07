import * as ciphs from "./encrpytion.js";
import dayjs from "dayjs";
import * as tus from "tus-js-client";

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

function bufferToBase64(buffer: ArrayBuffer): string {
    const bytes = new Uint8Array(buffer);
    let binary = "";
    for (let i = 0; i < bytes.byteLength; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return window.btoa(binary);
}
  
function Unit8ToBase64(uint8Array: Uint8Array<ArrayBuffer>): string {
    let binary = "";
    for (let i = 0; i < uint8Array.length; i++) {
      binary += String.fromCharCode(uint8Array[i]);
    }
    return window.btoa(binary);
}
  
// Retrieves public key from server and returns CryptoKey
async function getPublicKey() {
    const res = await fetch("/public");
    let text = await res.text();
    let key = await ciphs.transformRawPublicKey(text);
    return key;
}


export class Message {
    fileInput: HTMLInputElement;
    file: File;
    title: HTMLInputElement;
    vis: HTMLInputElement;
    expiry: HTMLInputElement;
    password: HTMLInputElement;
    content: HTMLInputElement;
    unlCheck: HTMLInputElement;
    downloadLimit: HTMLInputElement;
  
    dimension: Dimension;
    isEncrypted: Boolean;
    dimensionText: string;
    dimensionFileName: string;
  
    friendlyEncoder: TextEncoder;
    constructor(
        title:HTMLInputElement, vis:HTMLInputElement, expiry:HTMLInputElement,
        password:HTMLInputElement, content:HTMLInputElement, 
        fileInput:HTMLInputElement, unlCheck: HTMLInputElement, downloadLimit:HTMLInputElement
    ) {
      this.fileInput = fileInput;
  
      this.file = this.fileInput.files[0];
  
      this.title = title;
      this.vis = vis;
      this.expiry = expiry;
      this.password = password;
      this.content = content;
      this.unlCheck = unlCheck;
      this.downloadLimit = downloadLimit;


      this.dimension = {
        id: "",
        title: "",
        encrypted: false,
        visibility: false,
        fileName: "",
        text: "",
        downloadLimit: 0,
        expirationDate: 0,
        expirationDateISO: "",
      };
      this.isEncrypted = false;
      this.dimensionText = "";
      this.dimensionFileName = "";
  
      this.friendlyEncoder = new TextEncoder();
    }
    populateDimension() {
      let durMilli = dayjs.duration(this.expiry.value).asMilliseconds();
      let expirationDate = dayjs().add(durMilli, "milliseconds");
  
      let downloadLimitNum: number = 0;
      if (!this.unlCheck.checked) {
        downloadLimitNum = parseInt(this.downloadLimit.value);
      }
  
      this.dimension = {
        id: "",
        title: this.title.value,
        encrypted: this.isEncrypted,
        visibility: this.vis.checked,
        fileName: this.dimensionFileName,
        text: this.dimensionText,
        downloadLimit: downloadLimitNum,
        expirationDate: expirationDate.valueOf(),
        expirationDateISO: "",
      };
    }
    
    redirect(id: string) {
      window.location.href = "/grab?id=" + encodeURIComponent(id);
    }

    async uploadFile(id: string, token: string) {
      let blob: Blob;
      if (this.isEncrypted) {
        const iv = crypto.getRandomValues(new Uint8Array(12));
        const key = await ciphs.aesKeyFromPassword(this.password.value);
  
        const buffer = await this.file.arrayBuffer();
        const encrypted = await crypto.subtle.encrypt(
          { name: "AES-GCM", iv },
          key,
          buffer,
        );
        let mixture = ciphs.encryptedIVCombine(encrypted, iv);
        blob = new Blob([mixture]);
      } else {
        blob = this.file;
      }
  
      const upload = new tus.Upload(blob, {
        endpoint: "/files/",
        headers: {
          id: id,
          token: token,
        },
        onError: function (error) {
          console.log("Failed because: " + error);
        },
        onSuccess: function () {
          console.log("Download %s from %s", upload.file.name, upload.url);
        },
      });
  
      upload.start();
    }
  
    async encryptClientData() {
      this.dimensionText = await ciphs.base64encryptWithPassword(
        this.password.value,
        this.friendlyEncoder.encode(this.content.value),
      );
  
      if (this.file) {
        this.dimensionFileName = await ciphs.base64encryptWithPassword(
          this.password.value,
          this.friendlyEncoder.encode(this.file.name),
        );
      }
      this.isEncrypted = true;
    }
  
    async plainDimension() {
      this.dimensionText = this.content.value;
      if (this.file) {
        this.dimensionFileName = this.file.name;
      }
      this.isEncrypted = false;
    }
  
    // Hyprid encryption implementation, uses aes for bulk encryption and
    // encrypts the aes key with the public key(RSA) owned by the server
    async encryptPayload() {
      this.populateDimension();
      const serverPublicKey = await getPublicKey();
      
      let aes: CryptoKey, aesRaw: ArrayBuffer;
      ({ aes, aesRaw } = await ciphs.generateKeys());
  
      const rsaProtectedAES = await crypto.subtle.encrypt(
        { name: "RSA-OAEP" },
        serverPublicKey,
        aesRaw,
      );
  
      const iv = crypto.getRandomValues(new Uint8Array(12));
      const aesProDimension = await crypto.subtle.encrypt(
        {
          name: "AES-GCM",
          iv,
        },
        aes,
        new TextEncoder().encode(JSON.stringify(this.dimension)),
      );
  
      let combinedIVAndDimension = ciphs.encryptedIVCombine(aesProDimension, iv);
  
      return { rsaProtectedAES, combinedIVAndDimension };
    }
  
    async sendDimension(
      rsaProtectedAES: ArrayBuffer,
      combinedIVAndDimension: Uint8Array<ArrayBuffer>,
    ): Promise<Response> {
      let res = await fetch("api/enter", {
        method: "POST",
        headers: { "Content-Type": "application/json; charset=UTF-8" },
        body: JSON.stringify({
          aesKey: bufferToBase64(rsaProtectedAES),
          data: bufferToBase64(combinedIVAndDimension),
        }),
      });
      if (!res.ok) {
        throw new Error("post request insert failed");
      }
      return res;
    }
  }
  