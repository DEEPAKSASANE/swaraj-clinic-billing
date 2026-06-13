document.addEventListener("DOMContentLoaded", function () {
  const priceInput = document.getElementById("price");
  const discountInput = document.getElementById("discount");
  const totalField = document.getElementById("total");

  function updateTotal() {
    const price = parseFloat(priceInput.value) || 0;
    const discount = parseFloat(discountInput.value) || 0;
    const total = price - (price * discount / 100);
    totalField.value = total.toFixed(2);
  }

  priceInput.addEventListener("input", updateTotal);
  discountInput.addEventListener("input", updateTotal);

  
});