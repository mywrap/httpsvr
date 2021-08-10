"use strict"; // ES6


window.onload = () => {
    let paragraph0 = document.getElementById("paragraph0");
    let button0 = document.getElementById("button0");

    button0.addEventListener("click", () => {
        paragraph0.innerHTML = "changed content";
        console.log("did handle click on button0")
    });
};
