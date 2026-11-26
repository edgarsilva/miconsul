/**
 * Shows the picture chosen in a file input marked data-picture-preview="<selector>"
 * in place of the avatar inside the element that the selector matches.
 */
document.addEventListener("change", (event) => {
	const input = event.target;
	const isNotInput = !(input instanceof HTMLInputElement);
	const isNotPicturePreview = !input.matches('input[type="file"][data-picture-preview]');
	if (isNotInput || isNotPicturePreview) {
		return;
	}

	const file = input.files[0];
	const target = document.querySelector(input.dataset.picturePreview);
	const isImage = file?.type.startsWith("image/");
	if (!file || !target || !isImage) {
		return;
	}

	const reader = new FileReader();
	reader.addEventListener("load", () => {
		const avatar = target.querySelector(".avatar");
		if (!avatar) {
			return;
		}

		const image = document.createElement("img");
		image.src = reader.result;

		const frame = document.createElement("div");
		frame.className = "w-12 rounded-full";
		frame.append(image);

		const preview = document.createElement("div");
		preview.className = "avatar";
		preview.append(frame);

		avatar.replaceWith(preview);
	});
	reader.readAsDataURL(file);
});
