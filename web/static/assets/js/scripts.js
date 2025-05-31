function goToPage(e, action) {
    const url = new URL(window.location.href);
    let page = parseInt(url.searchParams.get("page")) || 1;

    if (action === "next") {
        page += 1;
    } else {
        page -= 1;
    }

    if (page < 1 || isNaN(page)) return;

    url.searchParams.set("page", page);
    window.location.href = url.toString();
}

function attachConfirmationToForms(className, options) {
    const buttons = document.querySelectorAll(className);
    buttons.forEach(btn => {
        btn.addEventListener('click', function (e) {
            e.preventDefault();
            const form = this.closest('form');

            Swal.fire({
                title: options.title || 'Are you sure?',
                text: options.text,
                icon: options.icon || 'warning',
                showCancelButton: true,
                confirmButtonColor: options.confirmButtonColor || '#3085d6',
                cancelButtonColor: options.cancelButtonColor || '#aaa',
                confirmButtonText: 'Yes, proceed!',
                reverseButtons: true
            }).then((result) => {
                if (result.isConfirmed) {
                    form.submit();
                }
            });
        });
    });
}

document.addEventListener('DOMContentLoaded', function () {
    attachConfirmationToForms('.delete-btn', {
        title: 'Are you sure?',
        text: 'This job will be removed.',
        icon: 'warning',
        confirmButtonColor: '#d33',
        confirmButtonText: 'Yes, delete it!'
    });

    attachConfirmationToForms('.change-status-btn', {
        title: 'Are you sure?',
        text: '',
        icon: 'warning',
        confirmButtonColor: '#d33',
        confirmButtonText: 'Yes, delete it!'
    });

    attachConfirmationToForms('.run-btn', {
        title: 'Run this job?',
        text: 'This job will be executed immediately.',
        icon: 'question',
        confirmButtonColor: '#3085d6',
        confirmButtonText: 'Yes, run it!'
    });
});


document.addEventListener('DOMContentLoaded', function () {
    const flashMessage = getCookie("flash");
    if (flashMessage) {
        Swal.fire({
            icon: 'success',
            title: 'Success',
            text: decodeURIComponent(flashMessage),
            confirmButtonColor: '#3085d6',
            confirmButtonText: 'OK'
        });
        document.cookie = "flash=; Max-Age=0; path=/";
    }

    const warningMessage = getCookie("warning");
    if (warningMessage) {
        Swal.fire({
            icon: 'warning',
            title: 'Warning',
            text: decodeURIComponent(warningMessage),
            confirmButtonColor: '#3085d6',
            confirmButtonText: 'OK'
        });
        document.cookie = "flash=; Max-Age=0; path=/";
    }

    function getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return (parts.pop().split(';').shift()).replaceAll('+', ' ');
    }
});