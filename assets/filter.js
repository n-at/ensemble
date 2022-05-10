(() => {

    $(() => {
        const $filterField = $('#filter');

        $filterField.keyup(() => {
            const inputValue = escapeRegExp($filterField.val());
            const regexp = new RegExp(inputValue, 'i');

            $('.filter-container')
                .find('.filter-element')
                .each((idx, el) => {
                    const $el = $(el);
                    const elementText = $el.find('.filter-field').text();

                    if (regexp.test(elementText)) {
                        $el.removeClass('d-none');
                    } else {
                        $el.addClass('d-none');
                    }
                })
        });
    });

    //https://stackoverflow.com/questions/3115150/how-to-escape-regular-expression-special-characters-using-javascript
    function escapeRegExp(text) {
        return text.replace(/[-[\]{}()*+?.,\\^$|#\s]/g, '\\$&');
    }

})();
