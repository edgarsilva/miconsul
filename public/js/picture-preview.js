/**
 * Shows the picture chosen in a file input marked data-picture-preview="<selector>"
 * in place of the avatar inside the element that the selector matches.
 */
document.addEventListener("change", (event) => {
  if (isPictureInput(event.target)) {
    previewPicture(event.target);
  }
});

function isPictureInput(element) {
  const isFileInput = element instanceof HTMLInputElement && element.type === "file";

  return isFileInput && element.hasAttribute("data-picture-preview");
}

function previewPicture(input) {
  const file = input.files[0];
  const target = document.querySelector(input.dataset.picturePreview);

  const hasTarget = target !== null;
  const isImage = file !== undefined && file.type.startsWith("image/");

  if (hasTarget && isImage) {
    readAsDataURL(file, (src) => showAvatar(target, src));
  }
}

function readAsDataURL(file, onLoad) {
  const reader = new FileReader();
  reader.addEventListener("load", () => onLoad(reader.result));
  reader.readAsDataURL(file);
}

function showAvatar(target, src) {
  const avatar = target.querySelector(".avatar");

  if (avatar) {
    avatar.replaceWith(newAvatar(src));
  }
}

function newAvatar(src) {
  const image = document.createElement("img");
  image.src = src;

  const frame = document.createElement("div");
  frame.className = "w-12 rounded-full";
  frame.append(image);

  const avatar = document.createElement("div");
  avatar.className = "avatar";
  avatar.append(frame);

  return avatar;
}
